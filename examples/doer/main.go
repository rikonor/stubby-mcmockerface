package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	// We'd like to make a network request
	url := "http://www.google.com"

	// 1. Using default http client
	n, err := FetchPageLengthBasic(url)
	if err != nil {
		fmt.Printf("Failed to fetch page %s length using basic method: %s\n", url, err)
	} else {
		fmt.Printf("Fetched page %s length using basic method: %d\n", url, n)
	}

	// 2. Using a given client
	c := &http.Client{
		Timeout: 3 * time.Second, // Just an example of how to create a non-default http client
	}

	n, err = FetchPageLengthUsingClient(c, url)
	if err != nil {
		log.Printf("Failed to fetch page %s length using given-client method: %s\n", url, err)
	} else {
		fmt.Printf("Fetched page %s length using given-client method: %d\n", url, n)
	}

	// 3. Using a HTTPClient/Doer interface
	// Lets mock an HTTPClient such that it always returns some test response
	mc := &MockHTTPClient{
		DoFn: func(req *http.Request) (*http.Response, error) {
			// Create our test ReadCloser (imitating http.Response.Body)
			r := ioutil.NopCloser(strings.NewReader("test response"))

			// return a mocked response
			return &http.Response{
				Body: r,
				// We can continue mocking as we please:
				// response status code, delay the response, etc
			}, nil
		},
	}

	n, err = FetchPageLengthUsingHTTPClient(mc, url)
	if err != nil {
		fmt.Printf("Failed to fetch page %s length using http-client method: %s\n", url, err)
	} else {
		fmt.Printf("Fetched page %s length using http-client method: %d\n", url, n)
	}

	// 4. We can augment HTTPClient and introduce Retry functionality into it
	rc := RetryHTTPClient(c, 3)

	n, err = FetchPageLengthUsingHTTPClient(rc, url)
	if err != nil {
		fmt.Printf("Failed to fetch page %s length using retry-client method: %s\n", url, err)
	} else {
		fmt.Printf("Fetched page %s length using retry-client method: %d\n", url, n)
	}
}

// FetchPageLengthBasic tries to retrieve the length of a page
func FetchPageLengthBasic(url string) (int, error) {
	// Using http.Get will fetch the URL using the default client from the http package
	res, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	// Read the response into memory and return a count of the byte slice size
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	return len(bs), nil
}

// FetchPageLengthUsingClient tries to retrieve the length of a page
// using a given http client
func FetchPageLengthUsingClient(c *http.Client, url string) (int, error) {
	// Build a GET request which we can feed to the given client later on
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// Use the given client to make the request
	res, err := c.Do(req)
	if err != nil {
		return 0, err
	}

	// Read the response into memory and return a count of the byte slice size
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	return len(bs), nil
}

// HTTPClient is a general interface for http clients
// Coincidentally, it is implemented by http.Client
// If we want to be more idiomatic it can also be named Doer
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MockHTTPClient is a mockable HTTPClient
type MockHTTPClient struct {
	DoFn func(req *http.Request) (*http.Response, error)
}

// Do calls the underlying Do method
func (c *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.DoFn(req)
}

// FetchPageLengthUsingHTTPClient is identical to the `UsingClient` example
// however it uses a general interface instead of an explicit http.Client
func FetchPageLengthUsingHTTPClient(c HTTPClient, url string) (int, error) {
	// Build a GET request which we can feed to the given client later on
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// Use the given client to make the request
	res, err := c.Do(req)
	if err != nil {
		return 0, err
	}

	// Read the response into memory and return a count of the byte slice size
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	return len(bs), nil
}

// FromString returns an HTTPClient which always returns a response with the given string
func FromString(s string) HTTPClient {
	return &MockHTTPClient{
		DoFn: func(req *http.Request) (*http.Response, error) {
			// convert the given string to a ReadCloser (same as Response.Body)
			body := ioutil.NopCloser(strings.NewReader(s))

			// Just return a response with the given string
			return &http.Response{
				Body: body,

				// Can mock other fields as well: StatusCode, etc
			}, nil
		},
	}
}

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
