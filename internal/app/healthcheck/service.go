package healthcheck

import (
	"context"
	"fmt"
	"francoggm/rinhabackend-2025-go-redis/internal/models"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
)

type ProcessorsHealth struct {
	Default  *models.HealthCheck
	Fallback *models.HealthCheck
}

const (
	leaderLockKey       = "processor_health_leader_lock"
	processorsHealthKey = "processors_health_status"
	leaderLockTTL       = 15 * time.Second
	routineInterval     = 5 * time.Second
)

type HealthCheckService struct {
	defaultUrl  string
	fallbackUrl string
	client      *fasthttp.Client
	cache       *redis.Client
	instanceID  string

	healthMutex sync.RWMutex
	processor   string
}

func NewHealthCheckService(defaultUrl, fallbackUrl string, cache *redis.Client) *HealthCheckService {
	service := &HealthCheckService{
		defaultUrl:  defaultUrl,
		fallbackUrl: fallbackUrl,
		client:      &fasthttp.Client{MaxConnsPerHost: 10},
		cache:       cache,
		instanceID:  uuid.New().String(),
	}

	go service.backgroundRoutine()
	return service
}

func (s *HealthCheckService) AvailableProcessor(ctx context.Context) string {
	s.healthMutex.RLock()
	processor := s.processor
	s.healthMutex.RUnlock()

	return processor
}

func (s *HealthCheckService) backgroundRoutine() {
	ticker := time.NewTicker(routineInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		acquire, err := s.cache.SetNX(ctx, leaderLockKey, s.instanceID, leaderLockTTL).Result()
		if err != nil {
			log.Printf("Error acquiring leader lock for instance %s: %v\n", s.instanceID, err)
			continue
		}

		isLeader := acquire
		if !isLeader {
			currentLeader, err := s.cache.Get(ctx, leaderLockKey).Result()
			if err == nil && currentLeader == s.instanceID {
				isLeader = true
			}
		}

		if isLeader {
			s.performChecksAndUpdate(ctx)

			if err := s.cache.Expire(ctx, leaderLockKey, leaderLockTTL).Err(); err != nil {
				log.Printf("Error renewing leader lock: %v\n", err)
			}
		}

		s.syncHealth(ctx)
	}
}

func (s *HealthCheckService) performChecksAndUpdate(ctx context.Context) {
	var wg sync.WaitGroup
	var defaultHealth, fallbackHealth *models.HealthCheck
	var defaultErr, fallbackErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		defaultHealth, defaultErr = s.checkHealth(s.defaultUrl)
	}()
	go func() {
		defer wg.Done()
		fallbackHealth, fallbackErr = s.checkHealth(s.fallbackUrl)
	}()
	wg.Wait()

	if defaultErr != nil {
		log.Println("Error checking default health:", defaultErr)
		return
	}

	if fallbackErr != nil {
		log.Println("Error checking fallback health:", fallbackErr)
		return
	}

	combinedHealth := &ProcessorsHealth{
		Default:  defaultHealth,
		Fallback: fallbackHealth,
	}

	payload, err := sonic.Marshal(combinedHealth)
	if err != nil {
		log.Println("Error marshalling combined health check:", err)
		return
	}

	if err := s.cache.Set(ctx, processorsHealthKey, payload, 30*time.Second).Err(); err != nil {
		log.Println("Error setting combined health in Redis:", err)
	}
}

func (s *HealthCheckService) checkHealth(url string) (*models.HealthCheck, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(url + "/payments/service-health")
	req.Header.SetMethod(http.MethodGet)
	req.Header.SetContentType("application/json")

	if err := s.client.DoTimeout(req, resp, 5*time.Second); err != nil {
		return nil, fmt.Errorf("failed to make payment request: %w", err)
	}

	statusCode := resp.StatusCode()
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("payment request failed with status code: %d", statusCode)
	}

	body := resp.Body()

	var healthCheck models.HealthCheck
	if err := sonic.Unmarshal(body, &healthCheck); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health check response: %w", err)
	}

	return &healthCheck, nil
}

func (s *HealthCheckService) syncHealth(ctx context.Context) {
	payload, err := s.cache.Get(ctx, processorsHealthKey).Bytes()
	if err == redis.Nil {
		return
	}
	if err != nil {
		log.Println("Error getting health status from Redis:", err)
		return
	}

	var healthStatus ProcessorsHealth
	if err := sonic.ConfigFastest.Unmarshal(payload, &healthStatus); err != nil {
		log.Println("Error unmarshalling health status from Redis:", err)
		return
	}

	processor := s.calculateProcessor(healthStatus)
	log.Printf("Calculated best processor: %s\n", processor)

	s.healthMutex.Lock()
	s.processor = processor
	s.healthMutex.Unlock()
}

func (s *HealthCheckService) calculateProcessor(healthStatus ProcessorsHealth) string {
	defaultHealth := healthStatus.Default
	fallbackHealth := healthStatus.Fallback

	if defaultHealth != nil && !defaultHealth.IsFailing && defaultHealth.MinResponseTime <= 300 {
		return "default"
	}

	if fallbackHealth != nil && !fallbackHealth.IsFailing && defaultHealth.MinResponseTime <= 200 {
		return "fallback"
	}

	if defaultHealth != nil && !defaultHealth.IsFailing {
		return "default"
	}

	return ""
}
