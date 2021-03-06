// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gin-contrib/sse"
	"github.com/go-ego/ego"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareGeneralCase(t *testing.T) {
	signature := ""
	router := ego.New()
	router.Use(func(c *ego.Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *ego.Context) {
		signature += "C"
	})
	router.GET("/", func(c *ego.Context) {
		signature += "D"
	})
	router.NoRoute(func(c *ego.Context) {
		signature += " X "
	})
	router.NoMethod(func(c *ego.Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "ACDB", signature)
}

func TestMiddlewareNoRoute(t *testing.T) {
	signature := ""
	router := ego.New()
	router.Use(func(c *ego.Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *ego.Context) {
		signature += "C"
		c.Next()
		c.Next()
		c.Next()
		c.Next()
		signature += "D"
	})
	router.NoRoute(func(c *ego.Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *ego.Context) {
		signature += "G"
		c.Next()
		signature += "H"
	})
	router.NoMethod(func(c *ego.Context) {
		signature += " X "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "ACEGHFDB", signature)
}

func TestMiddlewareNoMethodEnabled(t *testing.T) {
	signature := ""
	router := ego.New()
	router.HandleMethodNotAllowed = true
	router.Use(func(c *ego.Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *ego.Context) {
		signature += "C"
		c.Next()
		signature += "D"
	})
	router.NoMethod(func(c *ego.Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *ego.Context) {
		signature += "G"
		c.Next()
		signature += "H"
	})
	router.NoRoute(func(c *ego.Context) {
		signature += " X "
	})
	router.POST("/", func(c *ego.Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 405, w.Code)
	assert.Equal(t, "ACEGHFDB", signature)
}

func TestMiddlewareNoMethodDisabled(t *testing.T) {
	signature := ""
	router := ego.New()
	router.HandleMethodNotAllowed = false
	router.Use(func(c *ego.Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *ego.Context) {
		signature += "C"
		c.Next()
		signature += "D"
	})
	router.NoMethod(func(c *ego.Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *ego.Context) {
		signature += "G"
		c.Next()
		signature += "H"
	})
	router.NoRoute(func(c *ego.Context) {
		signature += " X "
	})
	router.POST("/", func(c *ego.Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "AC X DB", signature)
}

func TestMiddlewareAbort(t *testing.T) {
	signature := ""
	router := ego.New()
	router.Use(func(c *ego.Context) {
		signature += "A"
	})
	router.Use(func(c *ego.Context) {
		signature += "C"
		c.AbortWithStatus(401)
		c.Next()
		signature += "D"
	})
	router.GET("/", func(c *ego.Context) {
		signature += " X "
		c.Next()
		signature += " XX "
	})

	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "ACD", signature)
}

func TestMiddlewareAbortHandlersChainAndNext(t *testing.T) {
	signature := ""
	router := ego.New()
	router.Use(func(c *ego.Context) {
		signature += "A"
		c.Next()
		c.AbortWithStatus(410)
		signature += "B"

	})
	router.GET("/", func(c *ego.Context) {
		signature += "C"
		c.Next()
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 410, w.Code)
	assert.Equal(t, "ACB", signature)
}

// TestFailHandlersChain - ensure that Fail interrupt used middleware in fifo order as
// as well as Abort
func TestMiddlewareFailHandlersChain(t *testing.T) {
	// SETUP
	signature := ""
	router := ego.New()
	router.Use(func(context *ego.Context) {
		signature += "A"
		context.AbortWithError(500, errors.New("foo"))
	})
	router.Use(func(context *ego.Context) {
		signature += "B"
		context.Next()
		signature += "C"
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "A", signature)
}

func TestMiddlewareWrite(t *testing.T) {
	router := ego.New()
	router.Use(func(c *ego.Context) {
		c.String(400, "hola\n")
	})
	router.Use(func(c *ego.Context) {
		c.XML(400, ego.Map{"foo": "bar"})
	})
	router.Use(func(c *ego.Context) {
		c.JSON(400, ego.Map{"foo": "bar"})
	})
	router.GET("/", func(c *ego.Context) {
		c.JSON(400, ego.Map{"foo": "bar"})
	}, func(c *ego.Context) {
		c.Render(400, sse.Event{
			Event: "test",
			Data:  "message",
		})
	})

	w := performRequest(router, "GET", "/")

	assert.Equal(t, 400, w.Code)
	assert.Equal(t, strings.Replace("hola\n<map><foo>bar</foo></map>{\"foo\":\"bar\"}{\"foo\":\"bar\"}event:test\ndata:message\n\n", " ", "", -1), strings.Replace(w.Body.String(), " ", "", -1))
}
