module trading-platform/market-data-service

go 1.21

require (
    github.com/google/uuid v1.3.1
    github.com/gorilla/mux v1.8.0
    github.com/gorilla/websocket v1.5.0
    trading-platform/shared v0.0.0
)

replace trading-platform/shared => ../shared