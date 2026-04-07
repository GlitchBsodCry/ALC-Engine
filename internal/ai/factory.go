package ai

import (
	"context"
	"fmt"
	"sync"
)

type ModelCreator func(ctx context.Context) (AIModel, error)

type AIModelFactory struct {
	creators map[string]ModelCreator
}

var (
	globalFactory *AIModelFactory
	factoryOnce   sync.Once
)

func GetGlobalFactory() *AIModelFactory {
	factoryOnce.Do(func() {
		globalFactory = &AIModelFactory{
			creators: make(map[string]ModelCreator),
		}
		globalFactory.registerCreators()
	})
	return globalFactory
}

func (f *AIModelFactory) registerCreators() {
	f.creators["siliconflow"] = func(ctx context.Context) (AIModel, error) {
		return NewSiliconFlowModel(ctx)
	}

	f.creators["openai"] = func(ctx context.Context) (AIModel, error) {
		return NewSiliconFlowModel(ctx)
	}

	f.creators["ollama"] = func(ctx context.Context) (AIModel, error) {
		return NewSiliconFlowModel(ctx)
	}
}

func (f *AIModelFactory) CreateAIModel(ctx context.Context, modelType string) (AIModel, error) {
	creator, ok := f.creators[modelType]
	if !ok {
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
	return creator(ctx)
}

func (f *AIModelFactory) CreateAIHelper(ctx context.Context, modelType string, sessionID string) (*AIHelper, error) {
	model, err := f.CreateAIModel(ctx, modelType)
	if err != nil {
		return nil, err
	}
	return NewAIHelper(model, sessionID), nil
}

func (f *AIModelFactory) RegisterModel(modelType string, creator ModelCreator) {
	f.creators[modelType] = creator
}
