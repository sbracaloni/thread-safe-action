Thread Safe subscriber
======================


This is a dummy example of a subscription handler manager.

Users can create a subscription to an `ActivityTheme` providing their names and they obtain a `subscription ID`

Users can get the number of subscriptions by `ActivityTheme`.

Users can remove a subscription to an `ActivityTheme` providing their `subscription ID`.


- [Lock free subscribeHandler](sub/handler.go)
- [Usage/tests](subscriber_example_test.go)