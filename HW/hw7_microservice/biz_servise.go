package main

import (
	"context"
	"fmt"
)

type BizServerImpl struct {
	Alc     map[string][]string
	Storage []*Event
}

func (bs *BizServerImpl) Check(context.Context, *Nothing) (*Nothing, error) {
	fmt.Println("Do check")
	return &Nothing{}, nil
}

func (bs *BizServerImpl) Add(context.Context, *Nothing) (*Nothing, error) {
	fmt.Println("Do add")
	return &Nothing{}, nil
}

func (bs *BizServerImpl) Test(context.Context, *Nothing) (*Nothing, error) {
	fmt.Println("Do test")
	return &Nothing{}, nil
}
