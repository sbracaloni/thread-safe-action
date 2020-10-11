package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sbracaloni/thread-safe-action/action"

	"subscribers/sub"
)

func demoConcurrentCreateSubThenConcurrentDeleteSub(subHandler sub.SubscriptionHandler) {
	/*
		- Start 100 concurrent subscription creations
		- While creating keep asking fot he number of subscriptions by theme (concurrent reads and writes)
		- Start 100 concurrent subscription deletions

		The deletion starts only after all the subscriptions are done. the subCreatedChan has a buffer of nbUsers.
	*/
	fmt.Println("Start demo concurrent create and read, then delete all subscriptions")
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
	getCountUntilAllSubscribed(subHandler, len(randomSubToBeDone))

	// run subscription deletion concurrently
	concurrentDeleteSubscriptions(subHandler, subCreatedChan, nbUsers)

	// get number of subscriptions by theme until all subscriptions are removed
	getCountUntilAllSubscribed(subHandler, 0)
}

func demoConcurrentCreateAndDeleteSub(subHandler sub.SubscriptionHandler) {
	/*
		- Start 100 concurrent subscription creations and deletion as soon as it is created

		The subCreatedChan has no buffer and there is no synchronization wait before starting the deletion.
	*/
	fmt.Println("Start demo concurrent create and delete subscriptions")
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
	getCountUntilAllSubscribed(subHandler, 0)
}


func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	threadSafeHandler := action.NewThreadSafeActionHandler(ctx)
	subHandler := sub.NewSubscriptionHandlerLockFree(ctx, threadSafeHandler)
	demoConcurrentCreateSubThenConcurrentDeleteSub(subHandler)
	demoConcurrentCreateAndDeleteSub(subHandler)
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

func getCountUntilAllSubscribed(subHandler sub.SubscriptionHandler, nbExpectedSubscriptions int) {
	nbSubTotal := -1
	themeCount := map[sub.ActivityTheme]int{}
	for nbSubTotal != nbExpectedSubscriptions {
		nbSubTotal = 0
		for i := 0; i < 3; i++ {
			theme := sub.ActivityTheme(fmt.Sprintf("theme %d", i))
			count, err := subHandler.CountSubscriptionByTheme(theme)
			themeCount[theme] = count
			panicOnError(err)
			nbSubTotal += count
		}
		fmt.Println(fmt.Sprintf("Total subscriptions count: %d", nbSubTotal))
	}
	fmt.Println(fmt.Sprintf("Total subscriptions count reached: %d/%d", nbSubTotal, nbExpectedSubscriptions))
	for theme, count := range themeCount {
		fmt.Println(fmt.Sprintf("Total subscriptions for theme %s: %d", theme, count))

	}
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
