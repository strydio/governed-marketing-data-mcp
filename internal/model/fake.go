package model

import (
	"context"
	"fmt"
)

type FakeProvider struct {
	Responses []Response
	Calls     int
}

func (p *FakeProvider) Next(_ context.Context, _ Request) (Response, error) {
	if p.Calls >= len(p.Responses) {
		return Response{}, fmt.Errorf("fake provider has no response for call %d", p.Calls+1)
	}
	response := p.Responses[p.Calls]
	p.Calls++
	return response, nil
}
