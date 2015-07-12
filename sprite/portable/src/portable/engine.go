package main

import (
	"image"
	"image/color"
	"image/png"
	"io"

	"sync"

	"golang.org/x/mobile/event"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/sprite"
	"golang.org/x/mobile/exp/sprite/clock"
	"golang.org/x/mobile/exp/sprite/portable"
)

type WritableEngine struct {
	sync.RWMutex
	eng        sprite.Engine
	p          sprite.Engine
	img        *image.RGBA
	tx         map[sprite.Texture]sprite.Texture
	n          map[*sprite.Node]*sprite.Node
	clearColor color.Color
}

type arranger struct {
	e *WritableEngine
	a sprite.Arranger
}

func (ar *arranger) Arrange(e sprite.Engine, n *sprite.Node, t clock.Time) {
	ar.a.Arrange(ar.e, n, t)
}

func (e *WritableEngine) updateNode(n *sprite.Node) {
	if n.Arranger != nil {
		if _, ok := n.Arranger.(*arranger); !ok {
			n.Arranger = &arranger{e, n.Arranger}
		}
	}
	e.n[n].Parent = e.n[n.Parent]
	e.n[n].FirstChild = e.n[n.FirstChild]
	e.n[n].LastChild = e.n[n.LastChild]
	e.n[n].PrevSibling = e.n[n.PrevSibling]
	e.n[n].NextSibling = e.n[n.NextSibling]
}

func (e *WritableEngine) Register(n *sprite.Node) {
	e.eng.Register(n)
	e.n[n] = &sprite.Node{}
	e.p.Register(e.n[n])
	e.updateNode(n)
}

func (e *WritableEngine) Unregister(n *sprite.Node) {
	e.eng.Unregister(n)
	e.p.Unregister(e.n[n])
	e.updateNode(n)
	delete(e.n, n)
}

func (e *WritableEngine) LoadTexture(a image.Image) (sprite.Texture, error) {
	t, err := e.eng.LoadTexture(a)
	if err != nil {
		return nil, err
	}

	pt, err := e.p.LoadTexture(a)

	if err != nil {
		return nil, err
	}

	e.tx[t] = pt

	return t, nil
}

func (e *WritableEngine) SetSubTex(n *sprite.Node, x sprite.SubTex) {
	e.eng.SetSubTex(n, x)
	e.p.SetSubTex(e.n[n], sprite.SubTex{e.tx[x.T], x.R})
}

func (e *WritableEngine) SetTransform(n *sprite.Node, m f32.Affine) {
	e.eng.SetTransform(n, m)
	e.p.SetTransform(e.n[n], m)
}

func (e *WritableEngine) Render(scene *sprite.Node, t clock.Time, c event.Config) {
	e.Lock()
	for x := 0; x < e.img.Bounds().Max.X; x++ {
		for y := 0; y < e.img.Bounds().Max.Y; y++ {
			e.img.Set(x, y, e.clearColor)
		}
	}
	for n, _ := range e.n {
		e.updateNode(n)
	}
	e.eng.Render(scene, t, c)
	e.p.Render(e.n[scene], t, c)
	e.Unlock()
}

func (e *WritableEngine) WriteTo(w io.Writer) error {
	e.RLock()
	defer e.RUnlock()
	return png.Encode(w, e.img)
}

func NewWritableEngine(eng sprite.Engine, r image.Rectangle, c color.Color) *WritableEngine {
	img := image.NewRGBA(r)
	return &WritableEngine{
		eng:        eng,
		p:          portable.Engine(img),
		img:        img,
		tx:         make(map[sprite.Texture]sprite.Texture),
		n:          make(map[*sprite.Node]*sprite.Node),
		clearColor: c,
	}
}
