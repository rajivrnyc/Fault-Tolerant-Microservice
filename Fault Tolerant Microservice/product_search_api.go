package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

type QueryResult struct {
	Products     []Product `json:"products"`
	TotalFound   int       `json:"total_found"`
	SearchTime   string    `json:"search_time"`
	CheckedCount int       `json:"checked_request,omitempty"`
	TotalChecked int64     `json:"total_checked,omitempty"`
}

var (
	searchBulkhead     = make(chan struct{}, 50)
	failures           int64
	lastFailureTime    time.Time
	circuitOpen        int32
	cooldownPeriod     = 5 * time.Second
	failThreshold      = 100
	concurrentRequests int32
	loadLock           sync.Mutex
	maxConcurrent      int32 = 50
	products           sync.Map
	productList        []int
	checkTotal         int64
	numProducts        = 100000
	checksPerSearch    = 100
	maxSize            = 20
	brands             = []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"}
	categories         = []string{"Electronics", "Books", "Home", "Outdoors", "Clothes"}
)

func productGenerator() {
	for i := 0; i < numProducts; i++ {
		brand := brands[i%len(brands)]
		category := categories[i%len(categories)]
		p := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %s %d", brand, i),
			Category:    category,
			Description: fmt.Sprintf("Product Description %d", i),
			Brand:       brand,
		}
		products.Store(i, p)
		productList = append(productList, i)
	}

	log.Printf("%d Products generated\n", numProducts)
}

// Helper function to record failures for circuitBreaker
func recordFailure() {
	atomic.AddInt64(&failures, 1)
	lastFailureTime = time.Now()

	if atomic.LoadInt64(&failures) >= int64(failThreshold) {
		atomic.StoreInt32(&circuitOpen, 1)
	}
}

// Helper function to reset circuit breaker
func resetCircuit() {
	atomic.StoreInt64(&failures, 0)
	atomic.StoreInt32(&circuitOpen, 0)
}

func searchFunc(w http.ResponseWriter, r *http.Request) {
	// Circuit breaker implementation
	if atomic.LoadInt32(&circuitOpen) == 1 {
		if time.Since(lastFailureTime) < cooldownPeriod {
			http.Error(w, "Circuit Open", http.StatusServiceUnavailable)
			return
		}
		resetCircuit()
	}

	select {
	case searchBulkhead <- struct{}{}:
		defer func() { <-searchBulkhead }()
	default:
		http.Error(w, "Request overload", http.StatusServiceUnavailable)
		return
	}
	// Increment the concurrent request counter at start
	atomic.AddInt32(&concurrentRequests, 1)

	// Decrement it when the request finishes
	defer atomic.AddInt32(&concurrentRequests, -1)

	start := time.Now()
	q := strings.ToLower(r.URL.Query().Get("q"))
	debug := r.URL.Query().Get("debug") == "1" || strings.ToLower(r.URL.Query().Get("debug")) == "true"

	// How many products to check for this request
	n := min(checksPerSearch, len(productList))

	// If there are too many requests, this keeps the service from being overwhelmed and fails fast
	if atomic.LoadInt32(&concurrentRequests) > maxConcurrent {
		http.Error(w, "Server overloaded, try again later", http.StatusServiceUnavailable)
		return
	}

	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = rand.Intn(len(productList))
	}

	results := make([]Product, 0, maxSize)
	matches := 0

	for _, idx := range indices {
		id := productList[idx]
		val, ok := products.Load(id)
		if !ok {
			continue
		}
		p := val.(Product)
		if q != "" && (strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Category), q)) {
			matches++
			if len(results) < maxSize {
				results = append(results, p)
			}
		}
	}

	// Simulate 20% crashes to demonstrate partial failure
	if rand.Float32() < 0.2 {
		recordFailure()
		log.Println("Product search failed")
		// Make busy work
		dummy := 0
		for i := 0; i < 30_000_000; i++ {
			dummy += i % 7
		}
		// Lock resources while time is burned
		loadLock.Lock()
		time.Sleep(50 * time.Millisecond)
		loadLock.Unlock()

		http.Error(w, "Overload failure simulation", http.StatusInternalServerError)
		return
	}
	atomic.StoreInt64(&failures, 0)

	atomic.AddInt64(&checkTotal, int64(n))
	ct := atomic.LoadInt64(&checkTotal)

	elapsed := time.Since(start).Seconds()
	resp := QueryResult{
		Products:   results,
		TotalFound: matches,
		SearchTime: fmt.Sprintf("%.4fs", elapsed),
	}
	if debug {
		resp.CheckedCount = n
		resp.TotalChecked = ct
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           "Go Product Search Service running",
		"num_products":      numProducts,
		"checks_per_search": checksPerSearch,
	})
}

func main() {
	productGenerator()

	http.HandleFunc("/", healthHandler)
	http.HandleFunc("/products/search", searchFunc)

	log.Println("Starting Product API on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
