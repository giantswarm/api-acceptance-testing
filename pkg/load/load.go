package load

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// ProduceLoad creates load on an endpoint.
// It finishes when durationLimit or requestLimit is reached, whatever comes earlier.
// Stats are logged to a file every 10 seconds.
func ProduceLoad(endpointURL string, durationLimit time.Duration, requestLimit int) {
	startTime := time.Now()
	lastOutput := time.Now()
	endTime := startTime.Add(durationLimit)

	var successCount int64
	var errorCount int64

	f, err := os.OpenFile("load.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	for i := 0; i < requestLimit; i++ {
		if time.Now().After(endTime) {
			return
		}

		interval := time.Now().Sub(lastOutput)
		if interval >= 10*time.Second {
			numRequests := errorCount + successCount
			duration := interval.Seconds() / float64(numRequests)
			log.Printf("numRequests %d, successCount %d, errorCount %d, error rate: %.5f, average request duration: %.5f Sec", numRequests, successCount, errorCount, float64(errorCount)/float64(numRequests), duration)
			successCount = 0
			errorCount = 0
			lastOutput = time.Now()
		}

		resp, err := http.Get(endpointURL)
		if err != nil {
			errorCount++
			continue
		}

		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			errorCount++
			continue
		}

		successCount++
	}
}
