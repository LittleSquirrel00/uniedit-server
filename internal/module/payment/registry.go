package payment

import (
	"errors"
	"sync"

	"github.com/uniedit/server/internal/module/payment/provider"
)

// ProviderRegistry manages multiple payment providers.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]provider.Provider
	native    map[string]provider.NativePaymentProvider
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]provider.Provider),
		native:    make(map[string]provider.NativePaymentProvider),
	}
}

// Register registers a provider.
func (r *ProviderRegistry) Register(p provider.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p

	// Also register as native if applicable
	if np, ok := p.(provider.NativePaymentProvider); ok {
		r.native[p.Name()] = np
	}
}

// Get returns a provider by name.
func (r *ProviderRegistry) Get(name string) (provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, errors.New("provider not found: " + name)
	}
	return p, nil
}

// GetNative returns a native payment provider by name.
func (r *ProviderRegistry) GetNative(name string) (provider.NativePaymentProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.native[name]
	if !ok {
		return nil, errors.New("native provider not found: " + name)
	}
	return p, nil
}

// GetByMethod returns a provider for the given payment method.
func (r *ProviderRegistry) GetByMethod(method PaymentMethod) (provider.Provider, error) {
	switch method {
	case PaymentMethodCard:
		return r.Get("stripe")
	case PaymentMethodAlipay:
		return r.Get("alipay")
	case PaymentMethodWechat:
		return r.Get("wechat")
	default:
		return nil, errors.New("unsupported payment method: " + string(method))
	}
}

// GetNativeByMethod returns a native provider for the given payment method.
func (r *ProviderRegistry) GetNativeByMethod(method PaymentMethod) (provider.NativePaymentProvider, error) {
	switch method {
	case PaymentMethodAlipay:
		return r.GetNative("alipay")
	case PaymentMethodWechat:
		return r.GetNative("wechat")
	default:
		return nil, errors.New("not a native payment method: " + string(method))
	}
}

// List returns all registered provider names.
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
