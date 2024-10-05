package appconfig

import (
	"fmt"
	"strings"
)

type Context struct {
	Name     string `mapstructure:"name"`
	Registry string `mapstructure:"registry"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type Config struct {
	ImagesDirectory string    `mapstructure:"images_directory"`
	Contexts        []Context `mapstructure:"contexts"`
	CurrentContext  string    `mapstructure:"current_context"`
	Verbose         bool      `mapstructure:"verbose"`
}

func (c *Config) findCurrentContext() (*Context, error) {
	var currentContext *Context
	for _, ctx := range c.Contexts {
		if ctx.Name == c.CurrentContext {
			currentContext = &ctx
			break
		}
	}
	if currentContext == nil {
		return nil, fmt.Errorf("could not find current context")
	}
	return currentContext, nil
}

// Override takes a reference and overrides it by prepending the registry from the current context
func (c *Config) Override(ref string) string {
	currentContext, err := c.findCurrentContext()
	if err != nil {
		return ref
	}

	// If the reference is already fully qualified, return it as is
	if strings.Contains(ref, currentContext.Registry) {
		return ref
	}

	// Prepend the registry from the current context to the reference
	fullRef := fmt.Sprintf("%s/%s", currentContext.Registry, ref)
	return fullRef
}

func (c *Config) CurrentRegistry() string {
	currentContext, err := c.findCurrentContext()
	if err != nil {
		return ""
	}
	return currentContext.Registry
}
