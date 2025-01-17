/*
 * Copyright (c) 2008-2021, Hazelcast, Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License")
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hazelcast/hazelcast-go-client"
)

// lockAndIncrement locks the given key, reads the value from it and sets back the incremented value.
func lockAndIncrement(myMap *hazelcast.Map, key string, wg *sync.WaitGroup) {
	// Signal completion before this goroutine exits.
	defer wg.Done()
	intValue := int64(0)
	// Create a new unique lock context.
	lockCtx := myMap.NewLockContext(context.Background())
	// Lock the key.
	// The key cannot be unlocked without the same lock context.
	if err := myMap.Lock(lockCtx, key); err != nil {
		panic(err)
	}
	// Remember to unlock the key, otherwise it won't be accessible elsewhere.
	defer myMap.Unlock(lockCtx, key)
	// The same lock context, or a derived one from that lock context must be used,
	// otherwise the Get operation below will block.
	v, err := myMap.Get(lockCtx, key)
	if err != nil {
		panic(err)
	}
	// If v is not nil, then there's already a value for the key.
	if v != nil {
		intValue = v.(int64)
	}
	// Increment and set the value back.
	intValue++
	// The same lock context, or a derived one from that lock context must be used,
	// otherwise the Set operation below will block.
	if err = myMap.Set(lockCtx, key, intValue); err != nil {
		panic(err)
	}
}

func main() {
	const goroutineCount = 100
	const key = "counter"

	// Start the client with defaults
	ctx := context.TODO()
	client, err := hazelcast.StartNewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// Get a random map.
	rand.Seed(time.Now().Unix())
	mapName := fmt.Sprintf("sample-%d", rand.Int())
	myMap, err := client.GetMap(ctx, mapName)
	if err != nil {
		log.Fatal(err)
	}
	// Lock and increment the value stored in key for goroutineCount times.
	wg := &sync.WaitGroup{}
	wg.Add(goroutineCount)
	for i := 0; i < goroutineCount; i++ {
		go lockAndIncrement(myMap, key, wg)
	}
	// Wait for all goroutines to complete.
	wg.Wait()
	// Retrieve the final value.
	// A lock context is not needed, since the key is unlocked.
	if lastValue, err := myMap.Get(context.Background(), key); err != nil {
		panic(err)
	} else {
		fmt.Println("lastValue", lastValue)
	}
	// Shutdown client
	client.Shutdown(ctx)
}
