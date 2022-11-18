package controllers

import (
	"context"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Action interface {
	Execute(ctx context.Context) error
}

type CreateStatus struct {
	client.Client
	obj client.Object
}

func (c CreateStatus) Execute(ctx context.Context) error {
	err := c.Create(ctx, c.obj)
	if err != nil {
		return fmt.Errorf("create obj err %s", err)
	}
	return nil
}

type PatchStatus struct {
	client.Client
	original client.Object
	new      client.Object
}

func (p PatchStatus) Execute(ctx context.Context) error {
	if reflect.DeepEqual(p.original, p.new) {
		return nil
	}

	err := p.Status().Patch(ctx, p.new, client.MergeFrom(p.original))
	if err != nil {
		return fmt.Errorf("patch obj err %s", err)
	}
	return nil
}
