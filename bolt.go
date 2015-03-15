package bolt

import (
	"log"
	"net/http"
	"sync"
)

type (
	Bolt struct {
		Router                     *router
		handlers                   []HandlerFunc
		maxParam                   byte
		notFoundHandler            HandlerFunc
		methodNotAllowedHandler    HandlerFunc
		internalServerErrorHandler HandlerFunc
		pool                       sync.Pool
	}
	// Option is used to configure bolt. They are passed while creating a new
	// instance of bolt.
	Option      func(*Bolt)
	HandlerFunc func(*Context)
)

const (
	MIMEJSON = "application/json"

	HeaderAccept             = "Accept"
	HeaderContentDisposition = "Content-Disposition"
	HeaderContentLength      = "Content-Length"
	HeaderContentType        = "Content-Type"
)

var MethodMap = map[string]uint8{
	"CONNECT": 1,
	"DELETE":  2,
	"GET":     3,
	"HEAD":    4,
	"OPTIONS": 5,
	"PATCH":   6,
	"POST":    7,
	"PUT":     8,
	"TRACE":   9,
}

// New creates a bolt instance with options.
func New(opts ...Option) (b *Bolt) {
	b = &Bolt{
		maxParam: 5,
		notFoundHandler: func(c *Context) {
			http.Error(c.Response, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			c.Halt()
		},
		methodNotAllowedHandler: func(c *Context) {
			http.Error(c.Response, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			c.Halt()
		},
		internalServerErrorHandler: func(c *Context) {
			http.Error(c.Response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			c.Halt()
		},
	}

	// Set options
	for _, o := range opts {
		o(b)
	}

	b.Router = NewRouter(b)
	b.pool.New = func() interface{} {
		return &Context{
			Response: &response{},
			params:   make(Params, b.maxParam),
			store:    make(store),
			i:        -1,
			bolt:     b,
		}
	}

	return
}

// MaxParam returns an option to set the max path param allowed. Default is 5,
// good enough for many users.
func MaxParam(n uint8) Option {
	return func(b *Bolt) {
		b.maxParam = n
	}
}

// NotFoundHandler returns an option to set a custom NotFound hanlder.
func NotFoundHandler(h HandlerFunc) Option {
	return func(b *Bolt) {
		b.notFoundHandler = h
	}
}

// MethodNotAllowedHandler returns an option to set a custom MethodNotAllowed
// handler.
func MethodNotAllowedHandler(h HandlerFunc) Option {
	return func(b *Bolt) {
		b.methodNotAllowedHandler = h
	}
}

// InternalServerErrorHandler returns an option to set a custom
// InternalServerError handler.
func InternalServerErrorHandler(h HandlerFunc) Option {
	return func(b *Bolt) {
		b.internalServerErrorHandler = h
	}
}

// Use adds middleware(s) to the chain.
func (b *Bolt) Use(h ...HandlerFunc) {
	b.handlers = append(b.handlers, h...)
}

// Connect adds CONNECT route.
func (b *Bolt) Connect(path string, h ...HandlerFunc) {
	b.Handle("CONNECT", path, h)
}

// Delete adds DELETE route.
func (b *Bolt) Delete(path string, h ...HandlerFunc) {
	b.Handle("DELETE", path, h)
}

// Get adds GET route.
func (b *Bolt) Get(path string, h ...HandlerFunc) {
	b.Handle("GET", path, h)
}

// Head adds HEAD route.
func (b *Bolt) Head(path string, h ...HandlerFunc) {
	b.Handle("HEAD", path, h)
}

// Options adds OPTIONS route.
func (b *Bolt) Options(path string, h ...HandlerFunc) {
	b.Handle("OPTIONS", path, h)
}

// Patch adds PATCH route.
func (b *Bolt) Patch(path string, h ...HandlerFunc) {
	b.Handle("PATCH", path, h)
}

// Post adds POST route.
func (b *Bolt) Post(path string, h ...HandlerFunc) {
	b.Handle("POST", path, h)
}

// Put adds PUT route.
func (b *Bolt) Put(path string, h ...HandlerFunc) {
	b.Handle("PUT", path, h)
}

// Trace adds TRACE route.
func (b *Bolt) Trace(path string, h ...HandlerFunc) {
	b.Handle("TRACE", path, h)
}

// Handle adds method, path and handler to the router.
func (b *Bolt) Handle(method, path string, h []HandlerFunc) {
	h = append(b.handlers, h...)
	l := len(h)
	b.Router.Add(method, path, func(c *Context) {
		c.handlers = h
		c.l = l
		c.Next()
	})
}

// Static serves static files.
func (b *Bolt) Static(path, root string) {
	fs := http.StripPrefix(path, http.FileServer(http.Dir(root)))
	b.Get(path+"/*", func(c *Context) {
		fs.ServeHTTP(c.Response, c.Request)
	})
}

// ServeFile serves a file.
func (b *Bolt) ServeFile(path, file string) {
	b.Get(path, func(c *Context) {
		http.ServeFile(c.Response, c.Request, file)
	})
}

// Index serves index file.
func (b *Bolt) Index(file string) {
	b.ServeFile("/", file)
}

func (b *Bolt) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Find and execute handler
	h, c, s := b.Router.Find(r.Method, r.URL.Path)
	c.reset(rw, r)
	if h != nil {
		h(c)
	} else {
		if s == NotFound {
			b.notFoundHandler(c)
		} else if s == NotAllowed {
			b.methodNotAllowedHandler(c)
		}
	}
	b.pool.Put(c)
}

func (b *Bolt) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, b))
}

func (b *Bolt) Stop(addr string) {
	panic("implement it")
}