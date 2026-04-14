package ai

import (
	"context"
	"sync"

	"mygo_bangforai/api/errors"
)

type ModelCreator func(ctx context.Context, cfg map[string]interface{}) (AIModel, error)

type AIModelFactory struct {
	creators map[string]ModelCreator
}

var (
	globalFactory *AIModelFactory
	factoryOnce   sync.Once
)

func GetGlobalFactory() *AIModelFactory {
	factoryOnce.Do(func() {
		globalFactory = &AIModelFactory{creators: make(map[string]ModelCreator)}
		globalFactory.registerCreators()
	})
	return globalFactory
}

func str(cfg map[string]interface{}, key string) (string, bool) {
	if cfg == nil {
		return "", false
	}
	v, ok := cfg[key].(string)
	return v, ok && v != ""
}

func (f *AIModelFactory) registerCreators() {
	f.creators[ModelTypeSiliconflow] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		return NewSiliconFlowModel(ctx)
	}
	f.creators[ModelTypeOpenAI] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		return NewSiliconFlowModel(ctx)
	}
	f.creators[ModelTypeRAG] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		u, ok := str(cfg, "username")
		if !ok {
			return nil, errors.NewError(errors.InvalidParams, "rag requires username in cfg", "AIModelFactory.registerCreators")
		}
		return NewAliRAGModel(ctx, u)
	}
	f.creators[ModelTypeMCP] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		return NewMCPModel(ctx)
	}
	f.creators[ModelTypeOllama] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		baseURL, _ := str(cfg, "baseURL")
		modelName, ok := str(cfg, "modelName")
		if !ok {
			return nil, errors.NewError(errors.InvalidParams, "ollama requires modelName in cfg", "AIModelFactory.registerCreators")
		}
		return NewOllamaModel(ctx, baseURL, modelName)
	}
}

func (f *AIModelFactory) CreateAIModel(ctx context.Context, modelType string, cfg map[string]interface{}) (AIModel, error) {
	if modelType == "" {
		modelType = ModelTypeSiliconflow
	}
	creator, ok := f.creators[modelType]
	if !ok {
		return nil, errors.NewError(errors.InvalidParams, "unsupported model type: "+modelType, "AIModelFactory.CreateAIModel")
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	return creator(ctx, cfg)
}

func (f *AIModelFactory) CreateAIHelper(ctx context.Context, modelType string, sessionID string, cfg map[string]interface{}) (*AIHelper, error) {
	model, err := f.CreateAIModel(ctx, modelType, cfg)
	if err != nil {
		return nil, err
	}
	return NewAIHelper(model, sessionID), nil
}

func (f *AIModelFactory) RegisterModel(modelType string, creator ModelCreator) {
	f.creators[modelType] = creator
}
