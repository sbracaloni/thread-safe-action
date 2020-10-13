package main_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	action "github.com/sbracaloni/thread-safe-action"
	"gotest.tools/assert"

	"subscribers/sub"
)

func Test_shouldBeAbleToSubscribeAndCountSubscriptionConcurrentlyThenDelete(t *testing.T) {
	/*
		- Start 100 concurrent subscription creations
		- While creating keep asking for the number of subscriptions by theme (concurrent reads and writes)
		- Start 100 concurrent subscription deletions

		The deletion starts only after all the subscriptions are done. the subCreatedChan has a buffer of nbUsers.
	*/
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	threadSafeHandler := action.NewThreadSafeActionHandler(ctx)
	subHandler := sub.NewSubscriptionHandlerLockFree(ctx, threadSafeHandler)
	nbUsers := 100
	subCreatedChan := make(chan subCreatedInfo, nbUsers)
	defer close(subCreatedChan)

	// create subscription info to to subscribe 100 users to 3 different themes
	randomSubToBeDone := getRandomSubToBeDone(nbUsers)

	// run subscription creation concurrently. The subCreatedChan is used to collect the created subscription infos
	// returned by the subscription creation in each goroutine
	concurrentCreateSubscriptions(subHandler, randomSubToBeDone, subCreatedChan)

	// get number of subscriptions by theme until all subscriptions are registered - concurrent read and write
	// from the client point of view
	nbSubscriptions, err := getCountUntilAllSubscribed(subHandler, len(randomSubToBeDone))
	assert.NilError(t, err)
	assert.Equal(t, nbSubscriptions, nbUsers)

	// run subscription deletion concurrently
	concurrentDeleteSubscriptions(subHandler, subCreatedChan, nbUsers)

	// get number of subscriptions by theme until all subscriptions are removed
	nbSubscriptions, err = getCountUntilAllSubscribed(subHandler, 0)
	assert.NilError(t, err)
	assert.Equal(t, nbSubscriptions, 0)

}

func Test_shouldBeAbleToSubscribeAndUnsubscribeConcurrently(t *testing.T) {
	/*
		- Start 100 concurrent subscription creations and deletion as soon as it is created

		The subCreatedChan has no buffer and there is no synchronization wait before starting the deletion.
	*/
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	threadSafeHandler := action.NewThreadSafeActionHandler(ctx)
	subHandler := sub.NewSubscriptionHandlerLockFree(ctx, threadSafeHandler)

	subCreatedChan := make(chan subCreatedInfo)
	defer close(subCreatedChan)
	nbUsers := 100

	// create subscription info to to subscribe 100 users to 3 different themes
	randomSubToBeDone := getRandomSubToBeDone(nbUsers)

	// run subscription creation concurrently. The subCreatedChan is used to collect the created subscription infos
	// returned by the subscription creation in each goroutine
	concurrentCreateSubscriptions(subHandler, randomSubToBeDone, subCreatedChan)

	// run subscription deletion concurrently
	concurrentDeleteSubscriptions(subHandler, subCreatedChan, nbUsers)

	// get number of subscriptions by theme until all subscriptions are removed
	nbSubscriptions, err := getCountUntilAllSubscribed(subHandler, 0)
	assert.NilError(t, err)
	assert.Equal(t, nbSubscriptions, 0)

}

type subToBeDoneInfo struct {
	theme sub.ActivityTheme
	name  sub.PersonName
}

type subCreatedInfo struct {
	theme sub.ActivityTheme
	ID    sub.SubscriptionID
}

func getRandomSubToBeDone(nbUsers int) []subToBeDoneInfo {
	var subToBeDone []subToBeDoneInfo
	for i := 0; i < nbUsers; i++ {
		subToBeDone = append(subToBeDone, struct {
			theme sub.ActivityTheme
			name  sub.PersonName
		}{theme: sub.ActivityTheme(fmt.Sprintf("theme %d", i%3)), name: sub.PersonName(fmt.Sprintf("Name %d", i))})
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(subToBeDone), func(i, j int) { subToBeDone[i], subToBeDone[j] = subToBeDone[j], subToBeDone[i] })
	return subToBeDone
}

func concurrentCreateSubscriptions(subHandler sub.SubscriptionHandler, subs []subToBeDoneInfo, subIDsChan chan subCreatedInfo) {
	for _, subToCreate := range subs {
		go func(s subToBeDoneInfo) {
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			subID, err := subHandler.AddNewSubscription(s.theme, s.name)
			subIDsChan <- subCreatedInfo{
				theme: s.theme,
				ID:    subID,
			}
			panicOnError(err)
		}(subToCreate)
	}
}

func getCountUntilAllSubscribed(subHandler sub.SubscriptionHandler, nbExpectedSubscriptions int) (int, error) {
	nbSubTotal := -1
	themeCount := map[sub.ActivityTheme]int{}
	for nbSubTotal != nbExpectedSubscriptions {
		nbSubTotal = 0
		for i := 0; i < 3; i++ {
			theme := sub.ActivityTheme(fmt.Sprintf("theme %d", i))
			count, err := subHandler.CountSubscriptionByTheme(theme)
			if err != nil {
				return nbSubTotal, err
			}
			themeCount[theme] = count
			nbSubTotal += count
		}
		fmt.Println(fmt.Sprintf("Total subscriptions count: %d", nbSubTotal))
	}
	fmt.Println(fmt.Sprintf("Total subscriptions count reached: %d/%d", nbSubTotal, nbExpectedSubscriptions))
	for theme, count := range themeCount {
		fmt.Println(fmt.Sprintf("Total subscriptions for theme %s: %d", theme, count))
	}
	return nbSubTotal, nil
}

func concurrentDeleteSubscriptions(subHandler sub.SubscriptionHandler, subs <-chan subCreatedInfo, nbUsers int) {
	for i := 0; i < nbUsers; i++ {
		createdSub := <-subs
		go func(i sub.SubscriptionID, t sub.ActivityTheme) {
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			subHandler.RemoveSubscriptionAsync(t, i)
		}(createdSub.ID, createdSub.theme)
	}

}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
