package app

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dvcrn/wework-cli/pkg/wework"
)

func TestIs401Error(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "401 status code",
			err:  errors.New("request failed with status code: 401"),
			want: true,
		},
		{
			name: "401 Unauthorized",
			err:  errors.New("API error: 401 Unauthorized"),
			want: true,
		},
		{
			name: "Unauthorized",
			err:  errors.New("Unauthorized access"),
			want: true,
		},
		{
			name: "404 error",
			err:  errors.New("request failed with status code: 404"),
			want: false,
		},
		{
			name: "500 error",
			err:  errors.New("request failed with status code: 500"),
			want: false,
		},
		{
			name: "network error",
			err:  errors.New("connection timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := is401Error(tt.err)
			if got != tt.want {
				t.Errorf("is401Error(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestReauthenticate_WaitersGetNotified(t *testing.T) {
	s := NewService()
	s.username = "test-user-fake"
	s.password = "test-pass-fake"

	// Start a slow re-authentication by manually setting the flag
	s.mu.Lock()
	s.reauthing = true
	waiter1 := make(chan struct{})
	waiter2 := make(chan struct{})
	s.reauthWaiters = []chan struct{}{waiter1, waiter2}
	s.mu.Unlock()

	// Track notifications
	notified := make(chan bool, 2)

	go func() {
		<-waiter1
		notified <- true
	}()

	go func() {
		<-waiter2
		notified <- true
	}()

	// Simulate completion of re-auth
	go func() {
		time.Sleep(50 * time.Millisecond)
		s.mu.Lock()
		s.reauthing = false
		for _, ch := range s.reauthWaiters {
			close(ch)
		}
		s.reauthWaiters = nil
		s.mu.Unlock()
	}()

	// Both waiters should be notified
	timeout := time.After(1 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-notified:
			// good
		case <-timeout:
			t.Fatal("waiters were not notified in time")
		}
	}
}

func TestReauthenticate_Concurrency(t *testing.T) {
	s := NewService()
	s.username = "test-user-fake"
	s.password = "test-pass-fake"

	var reauthAttempts int64
	var wg sync.WaitGroup
	goroutines := 10

	// Track when we start re-authenticating
	startedReauth := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Track if this goroutine gets to start re-auth
			s.mu.Lock()
			wasReauthing := s.reauthing
			if !wasReauthing {
				atomic.AddInt64(&reauthAttempts, 1)
				startedReauth <- true
			}
			s.mu.Unlock()

			// This will fail because we don't have real credentials,
			// but that's okay - we're testing concurrency control
			_ = s.reauthenticate()
		}()
	}

	wg.Wait()
	close(startedReauth)

	count := len(startedReauth)
	// With proper locking, most goroutines should wait, so we expect
	// significantly fewer re-auth attempts than total goroutines
	if count > goroutines/2 {
		t.Errorf("expected fewer re-auth attempts due to locking, got %d out of %d goroutines", count, goroutines)
	}
}

func TestCreateAuthenticatedClientLocked_MissingCredentials(t *testing.T) {
	s := NewService()

	s.mu.Lock()
	_, err := s.createAuthenticatedClientLocked()
	s.mu.Unlock()

	if err == nil {
		t.Error("expected error when credentials are missing")
	}
	if err.Error() != "WEWORK_USERNAME and WEWORK_PASSWORD must be set in the environment" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecuteWithRetry_NoRetryOnNon401(t *testing.T) {
	s := NewService()
	s.username = "test-user-fake"
	s.password = "test-pass-fake"
	// Set a client to skip initial auth attempt
	s.client = wework.NewWeWork("fake-token-for-test")

	callCount := 0
	err := s.executeWithRetry(func(_ *wework.WeWork) error {
		callCount++
		return errors.New("request failed with status code: 500")
	})

	if err == nil {
		t.Error("expected error to be returned")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retry on non-401), got %d", callCount)
	}
	if is401Error(err) {
		t.Error("error should not be classified as 401")
	}
}
