package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/rpc"
	"os"
	"sync"
)

const saveQueueLength = 1024

type Store interface {
	Put(url, key *string) error
	Get(key, url *string) error
}

type URLStore struct {
	mu   sync.RWMutex
	urls map[string]string
	save chan record
}

type record struct {
	Key, URL string
}

type ProxyStore struct {
	*URLStore // local cache
	client    *rpc.Client
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls: make(map[string]string),
	}

	if filename != "" {
		s.save = make(chan record, saveQueueLength)

		if err := s.load(filename); err != nil {
			log.Fatal("Error loading URLStore:", err)
		}

		go s.saveLoop(filename)
	}

	return s
}

func (s *URLStore) Get(key, url *string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if u, ok := s.urls[*key]; ok {
		*url = u
		return nil
	}

	return errors.New("key not found")
}

func (s *URLStore) Set(key, url *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, present := s.urls[*key]; present {
		return errors.New("key already exists")
	}

	s.urls[*key] = *url
	return nil
}

func (s *URLStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.urls)
}

func (s *URLStore) Put(url, key *string) error {
	for {
		*key = genKey(s.Count())
		if err := s.Set(key, url); err == nil {
			break
		}
	}

	if s.save != nil {
		s.save <- record{*key, *url}
	}

	return nil
}

func (s *URLStore) load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		return nil
	}
	defer file.Close()

	dec := json.NewDecoder(file)

	for {
		var r record
		switch err := dec.Decode(&r); err {
		case nil:
			s.Set(&r.Key, &r.URL)
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

func (s *URLStore) saveLoop(filename string) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Error opening URLStore:", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	for {
		// taking a record from the channel and encoding it
		r, ok := <-s.save
		if !ok {
			break
		}

		if err := enc.Encode(r); err != nil {
			log.Println("Error saving to URLStore:", err)
		}
	}
}

func NewProxyStore(addr string) *ProxyStore {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatalln("Error constructing ProxyStore:", err)
	}

	return &ProxyStore{URLStore: NewURLStore(""), client: client}
}

func (s *ProxyStore) Get(key, url *string) error {
	if err := s.URLStore.Get(key, url); err == nil {
		return nil
	}

	// rpc call to master
	if err := s.client.Call("Store.Get", key, url); err != nil {
		return err
	}

	s.URLStore.Set(key, url) // update local cache
	return nil
}

func (s *ProxyStore) Put(url, key *string) error {
	// rpc call to master
	if err := s.client.Call("Store.Put", url, key); err != nil {
		return err
	}

	s.URLStore.Set(key, url) // update local cache
	return nil
}
