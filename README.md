Stubby McMockerface
---

The goal of this document is to make a case for using Go interfaces for:
* dependency injection (DI)
* Easily mocking interfaces
* Augmenting an existing instance of an interface

### 1. Original Motivation - Dependency Injection (DI)

Dependency Injection is a Software Development technique whereby one object supplies the dependencies of another object. This is in contrast to the object building or finding it's dependencies from some global scope.

Lets look at a simple example.
We have a person that would like to introduce himself.

```go
type Person struct {
  Name string
}

func (p *Person) IntroduceYourself() {
  fmt.Println("Hi, my name is " + p.Name + ".")
}
```

We actually want to support different kinds of people, e.g a loud person, a normal person and perhaps a mute person.

```go
// say is how a normal person would speak
func say(msg string) {
  fmt.Println(msg)
}

// sayLoud is how a loud person would speak
func sayLoud(msg string) {
  fmt.Println(strings.ToUpper(msg))
}

// sayMute is how a mute person would speak
func sayMute(msg string) {
  // Do nothing because you're mute
}
```

In this case it might be a good idea us to allow Person to have a Say function injected into it, rather then having one tightly coupled implementation.

```go
// Define a type for Say functions
type SayFunc func(msg string)

// Make sure Person allows us to specify a Say function for it to use
type Person struct {
  ...
  SayFn SayFunc
}

// Make the person use the injected SayFn to introduce itself
func (p *Person) IntroduceYourself() {
  p.SayFn("Hi, my name is " + p.Name + ".")
}
```

```go
p := &Person{
  Name: "Kip",
  SayFn: SayFunc(sayLoud), // Inject a Say function of our choice
}

p.IntroduceYourself()

// Output: HI, MY NAME IS KIP.
```

### 2. Mocking via DI

A common issue when testing software is dealing with external services that our code depends on.
Having to depend on the availability of these services can complicate tests significantly.
A few examples are disk resources, network resources, databases, API libraries and more.

Let's examine how to solve this issue by mocking a service and providing the mocked version via dependency injection.
In this way, the code receiving the injected dependency doesn't even know that it did not receive the real service.

##### Example - `http.Client`

For our use-case we will examine `http.Client` which is a struct from the `net/http` standard Go lib which allows us to make network requests. The way we make a network request is by building an `http.Request` first and then providing it to the client to perform.

```go
// Make a GET request to `url`
req, err := http.NewRequest("GET", url, nil)

// Perform the request
res, err := http.DefaultClient.Do(req)
```

`http.Client` does not satisfy any specific interface out of the box, but that doesn't mean we can't create an interface to match it.

```go
// HTTPClient is a general interface for http clients
// Coincidentally, it is implemented by http.Client
// If we want to be more idiomatic it can also be named Doer
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

Now we'll see a pattern that can be used to very generically mock an interface.

```go
// MockHTTPClient is a mockable HTTPClient
type MockHTTPClient struct {
	DoFn func(req *http.Request) (*http.Response, error)
}

// Do calls the underlying Do method
func (c *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.DoFn(req)
}
```

`MockHTTPClient` takes in a generic `DoFn`, so we can create many different mock clients.

```go
// FromString returns an HTTPClient which always returns a response with the given string
func FromString(s string) HTTPClient {
	return &MockHTTPClient{
		DoFn: func(req *http.Request) (*http.Response, error) {
      // convert the given string to a ReadCloser (same as Response.Body)
      body := ioutil.NopCloser(strings.NewReader(s))

      // Just return a response with the given string
      return &http.Response{
        Body: body,

        // NOTE: We can mock other fields as well: StatusCode, cookies, headers, etc
      }, nil
		},
	}
}
```

Notice there are lots of options for mocking `HTTPClient`: `FromStatusCode`, `FromCookies`, `FromHeaders`, etc. We can also just create a `MockHTTPClient` on the fly with some special custom logic.

Let's say we have a function `FetchSomeNetworkResources`, which accepts an `HTTPClient`.
Although, under normal circumstances we would give it an `http.Client`, during tests we can just give it one of our mock clients.

```go
c := FromString("data: boop")

data, err := FetchSomeNetworkResources(c)
if data != "boop" {
  t.Fatal(...) // fail the test
}
```

We can follow the same pattern to mock any interface.

### 3. Enhancing existing interfaces

Another cool aspect of this pattern is that it can be used for much more then just mocking.
Note: It's been debated that in the context of "enhancing/augmenting" `StubXXX` is more appropriate then `MockXXX`. That said, we will keep using `MockXXX` for the sake of demonstration.

##### Example - `RetryHTTPClient`

```go
// RetryHTTPClient wraps an HTTPClient with retry functionality
func RetryHTTPClient(c HTTPClient, retries int) HTTPClient {
	return &MockHTTPClient{
		DoFn: func(req *http.Request) (*http.Response, error) {
			var res *http.Response
			var err error

			// try `retries` times
			for i := 0; i < retries; i++ {
				// attempt the request
				res, err = c.Do(req)
				if err != nil {
					// retry on failure
					continue
				}

				return res, nil
			}

			// we made `retries` attempts and never succeeded
			return nil, err
		},
	}
}
```

##### Example - `RewriteHostHTTPClient`

```go
// RewriteHostHTTPClient will rewrite the host of any request passing through it
func RewriteHostHTTPClient(c HTTPClient, host string) HTTPClient {
	return &MockHTTPClient{
		DoFn: func(req *http.Request) (*http.Response, error) {
			// Rewrite the Host portion of the request
			req.Host = host
			req.URL.Host = host

			// Send the request
			return c.Do(req)
		},
	}
}
```

##### Example - `Publisher` and friends

Let's define a `Publisher` interface.

```go
type Publisher interface {
  Publish(msg Message) error
}
```

And a generic mock publisher.

```go
type MockPublisher struct {
  PublishFn func(msg Message) error
}

func (p *MockPublisher) Publish(msg Message) error {
  return p.PublishFn(msg)
}
```

##### `TransformPublisher`

_Definition_

```go
// TransformFunc is a function that changes a message and returns the changed version
type TransformFunc func(msg string) string

// TransformPublisher wraps a given Publisher with a message TransformFunc
func TransformPublisher(p Publisher, tfn TransformFunc) Publisher {
	return &MockPublisher{
		PublishFn: func(msg string) error {
			// transform the message using the given transform function, then send it along
			return p.Publish(tfn(msg))
		},
	}
}
```

_Usage_

```go
// Lets try and create a Publisher that will transform our messages before sending them out
tp := TransformPublisher(p, func(msg string) string {
  // as an example, lets capitalize the message
  return strings.Title(msg)
})

// Should publish: "Hello"
tp.Publish("hello")
```

##### `MultiPublisher`

_Definition_

```go
// MultiPublisher wraps all given Publishers into one Publisher
func MultiPublisher(ps ...Publisher) Publisher {
	return &MockPublisher{
		PublishFn: func(msg string) error {
			// iterate over all publishers and send to each in turn
			for _, p := range ps {
				// there's multiple possible error handling strategies here
				// in this case we'll just return the first encountered error
				if err := p.Publish(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
```

_Usage_

```go
mp := MultiPublisher(p1, p2, p3)

// Should publish to `p1`, `p2` and `p3`
mp.Publish("hello")
```

##### `BatchPublisher`

_Definition_

```go
// BatchPublisher batches messages together before sending them out
func BatchPublisher(p Publisher, batchSize int) Publisher {
	// hold our batched msgs somewhere
	msgs := []string{}

	return &MockPublisher{
		PublishFn: func(msg string) error {
			msgs = append(msgs, msg)

			// if enough messages have been batched, we can send them out
			if len(msgs) == batchSize {
				// there's multiple ways to batch the messages
				// in this case we'll just concatenate them
				batchMsg := strings.Join(msgs, ",")
				return p.Publish(batchMsg)
			}

			// Note: It's also possible to flush the batch publisher after some pre-defined time duration
			// but to keep the example simple we will not do so

			// still waiting for batch buffer to fill up
			return nil
		},
	}
}
```

_Usage_

```go
bp := BatchPublisher(p, 3)

bp.Publish(msg) // Won't publish yet
bp.Publish(msg) // Won't publish yet
bp.Publish(msg) // Will publish all three now
```
