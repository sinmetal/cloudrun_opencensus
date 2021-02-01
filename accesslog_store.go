package main

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
)

var _ datastore.PropertyLoadSaver = &AccessLog{}

type AccessLog struct {
	ID        string `datastore:"-"`
	CreatedAt time.Time
}

// Load is implementation for PropertyLoadSaver.
func (entity *AccessLog) Load(ps []datastore.Property) error {
	err := datastore.LoadStruct(entity, ps)
	if err != nil {
		return err
	}

	return nil
}

// Save is implementation for PropertyLoadSaver.
func (entity *AccessLog) Save() ([]datastore.Property, error) {
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = time.Now()
	}

	ps, err := datastore.SaveStruct(entity)
	if err != nil {
		return nil, err
	}
	return ps, nil
}

type AccessLogStore struct {
	ds *datastore.Client
}

func NewAccessLogStore(ctx context.Context, ds *datastore.Client) (*AccessLogStore, error) {
	return &AccessLogStore{
		ds: ds,
	}, nil
}

func (s *AccessLogStore) Kind() string {
	return "AccessLog"
}

func (s *AccessLogStore) Key(id string) *datastore.Key {
	return datastore.NameKey(s.Kind(), id, nil)
}

func (s *AccessLogStore) Insert(ctx context.Context, log *AccessLog) (*datastore.Key, error) {
	keys, err := s.ds.Mutate(ctx, datastore.NewInsert(s.Key(log.ID), log))
	if err != nil {
		return nil, fmt.Errorf("failed insert AccessLog id=%s : %w", log.ID, err)
	}
	return keys[0], nil
}
