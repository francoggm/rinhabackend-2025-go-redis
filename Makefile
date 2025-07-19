build-run:
	docker compose up --build -d

down-vol:
	docker compose down --volumes

down:
	docker compose down
