package services

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

	healthMutex    sync.RWMutex
	inMemoryHealth *ProcessorsHealth
}

func NewHealthCheckService(defaultUrl, fallbackUrl string, cache *redis.Client) *HealthCheckService {
	service := &HealthCheckService{
		defaultUrl:     defaultUrl,
		fallbackUrl:    fallbackUrl,
		client:         &fasthttp.Client{},
		cache:          cache,
		instanceID:     uuid.New().String(),
		inMemoryHealth: &ProcessorsHealth{},
	}

	go service.backgroundRoutine()
	return service
}

func (s *HealthCheckService) DefaultHealthCheck(ctx context.Context) (*models.HealthCheck, error) {
	s.healthMutex.RLock()
	defer s.healthMutex.RUnlock()

	if s.inMemoryHealth == nil || s.inMemoryHealth.Default == nil {
		return nil, fmt.Errorf("default health status is not yet available")
	}

	return s.inMemoryHealth.Default, nil
}

func (s *HealthCheckService) FallbackHealthCheck(ctx context.Context) (*models.HealthCheck, error) {
	s.healthMutex.RLock()
	defer s.healthMutex.RUnlock()

	if s.inMemoryHealth == nil || s.inMemoryHealth.Fallback == nil {
		return nil, fmt.Errorf("fallback health status is not yet available")
	}

	return s.inMemoryHealth.Fallback, nil
}

func (s *HealthCheckService) backgroundRoutine() {
	ticker := time.NewTicker(routineInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		isLeader, err := s.tryAcquireLeader(ctx)
		if err != nil {
			log.Printf("Error acquiring leader lock for instance %s: %v\n", s.instanceID, err)
		}

		if isLeader {
			s.performChecksAndUpdate(ctx)
			log.Printf("Leader acquired for instance %s, performing health checks\n", s.instanceID)
		}

		s.syncHealth(ctx)
	}
}

func (s *HealthCheckService) tryAcquireLeader(ctx context.Context) (bool, error) {
	return s.cache.SetNX(ctx, leaderLockKey, s.instanceID, leaderLockTTL).Result()
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
	}
	if fallbackErr != nil {
		log.Println("Error checking fallback health:", fallbackErr)
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
	log.Printf("Syncing health status for instance %s: Default: %+v Fallback: %+v\n", s.instanceID, healthStatus.Default, healthStatus.Fallback)

	s.healthMutex.Lock()
	s.inMemoryHealth = &healthStatus
	s.healthMutex.Unlock()
}
