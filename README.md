# rinhabackend-2025-go-redis

This is a solution for the [Rinha de Backend 2025 challenge](https://github.com/zanfranceschi/rinha-de-backend-2025/tree/main).

The challenge is to create a service that processes payments through two different processors: a **default** processor with lower fees and a **fallback** processor with higher fees. The main goal is to process as many transactions as possible in the shortest amount of time, while prioritizing the lower-cost processor.

-----

## Solution Overview

This solution is built with **Golang** and leverages several patterns and technologies to meet the challenge's requirements:

* **Concurrency with Go Channels**: The application uses Go channels as a queuing system to handle a high volume of incoming payment requests concurrently. A pool of workers processes these payments, ensuring high throughput.

* **Redis for Caching and Persistence**: Redis is used for both caching health check statuses of the payment processors and for persisting payment data. This ensures that the application can quickly determine the best processor to use and that no payment data is lost.

* **Health Checking and Failover**: A background routine continuously checks the health of both the default and fallback processors. It assesses their response times and failure rates to dynamically determine which processor is the most viable option at any given moment. If the default processor becomes slow or unresponsive, the system automatically fails over to the fallback processor.

* **Load Balancing**: The architecture includes an NGINX load balancer to distribute traffic between multiple instances of the application server, enhancing scalability and availability.

-----

## How to Run

To run the project, you need to have Docker and Docker Compose installed.

1. Clone the repository.
2. Navigate to the project's root directory.
3. Run the following command to build and start all the services in the background:

    ```sh
    make build-run
    ```

    Alternatively, you can use docker-compose directly:

    ```sh
    docker-compose up --build -d
    ```

    This will start the application servers, the Redis instance, and the NGINX load balancer.

To stop the services and remove the containers, you can run:

```sh
make down
```

To stop the services and also remove the data volumes, run:

```sh
make down-vol
```

