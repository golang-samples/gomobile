package main

import (
	"fmt"
	"image"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"image/color"
	"image/png"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/sprite"
	"golang.org/x/mobile/exp/sprite/clock"
	"golang.org/x/mobile/exp/sprite/glsprite"
	"golang.org/x/mobile/exp/sprite/portable"
	"golang.org/x/mobile/gl"
)

type arrangerFunc func(e sprite.Engine, n *sprite.Node, t clock.Time)

func (a arrangerFunc) Arrange(e sprite.Engine, n *sprite.Node, t clock.Time) { a(e, n, t) }

func main() {
	app.Main(func(a app.App) {
		var c event.Config
		engs := []sprite.Engine{
			glsprite.Engine(),
			newEngine(),
		}
		roots := make(map[sprite.Engine]*sprite.Node, len(engs))
		startClock := time.Now()
		for e := range a.Events() {
			switch e := event.Filter(e).(type) {
			case event.Lifecycle:
				switch e.Crosses(event.LifecycleStageVisible) {
				case event.ChangeOn:
					for _, eng := range engs {
						ts := loadTextures(eng)
						roots[eng] = loadScene(eng, ts)
					}
				}
			case event.Config:
				c = e
			case event.Draw:
				now := clock.Time(time.Since(startClock) * 60 / time.Second)
				gl.ClearColor(1, 1, 1, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT)
				gl.Enable(gl.BLEND)
				gl.BlendEquation(gl.FUNC_ADD)
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				for eng, root := range roots {
					eng.Render(root, now, c)
				}
				a.EndDraw()
			}
		}
	})
}

type engine struct {
	sync.RWMutex
	eng sprite.Engine
	img *image.RGBA
}

func (e *engine) Register(n *sprite.Node) {
	e.Lock()
	e.eng.Register(n)
	e.Unlock()
}

func (e *engine) Unregister(n *sprite.Node) {
	e.Lock()
	e.eng.Unregister(n)
	e.Unlock()
}

func (e *engine) LoadTexture(a image.Image) (sprite.Texture, error) {
	e.Lock()
	defer e.Unlock()
	return e.eng.LoadTexture(a)
}

func (e *engine) SetSubTex(n *sprite.Node, x sprite.SubTex) {
	e.Lock()
	e.eng.SetSubTex(n, x)
	e.Unlock()
}

func (e *engine) SetTransform(n *sprite.Node, m f32.Affine) {
	e.Lock()
	e.eng.SetTransform(n, m)
	e.Unlock()

}

func (e *engine) Render(scene *sprite.Node, t clock.Time, c event.Config) {
	e.Lock()
	for x := 0; x < e.img.Bounds().Max.X; x++ {
		for y := 0; y < e.img.Bounds().Max.Y; y++ {
			e.img.Set(x, y, color.White)
		}
	}
	e.eng.Render(scene, t, c)
	e.Unlock()
}

func newEngine() sprite.Engine {
	img := image.NewRGBA(image.Rect(0, 0, 800, 800))
	eng := &engine{eng: portable.Engine(img), img: img}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		eng.RLock()
		defer eng.RUnlock()
		if err := png.Encode(w, img); err != nil {
			log.Println(err)
		}
	})

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				log.Println("Addr", ipnet.IP.String())
			}
		}
	}

	go http.ListenAndServe(":8080", nil)

	return eng
}

func loadScene(eng sprite.Engine, ts map[string]sprite.SubTex) *sprite.Node {
	root := &sprite.Node{}
	eng.Register(root)
	eng.SetTransform(root, f32.Affine{
		{1, 0, 0},
		{0, 1, 0},
	})

	n := &sprite.Node{}
	eng.Register(n)
	root.AppendChild(n)
	eng.SetTransform(n, f32.Affine{
		{200, 0, 0},
		{0, 200, 0},
	})
	n.Arranger = arrangerFunc(func(eng sprite.Engine, n *sprite.Node, t clock.Time) {
		s := fmt.Sprintf("%02d", int(t))
		eng.SetSubTex(n, ts[s[0:1]])
	})

	return root
}

func loadTextures(eng sprite.Engine) map[string]sprite.SubTex {
	a, err := asset.Open("tx_letters.png")
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	img, _, err := image.Decode(a)
	if err != nil {
		log.Fatal(err)
	}
	t, err := eng.LoadTexture(img)
	if err != nil {
		log.Fatal(err)
	}

	return map[string]sprite.SubTex{
		":":     sprite.SubTex{t, image.Rect(0, 0, 200, 200)},
		"0":     sprite.SubTex{t, image.Rect(200, 0, 400, 200)},
		"1":     sprite.SubTex{t, image.Rect(400, 0, 600, 200)},
		"2":     sprite.SubTex{t, image.Rect(0, 200, 200, 400)},
		"3":     sprite.SubTex{t, image.Rect(200, 200, 400, 400)},
		"4":     sprite.SubTex{t, image.Rect(400, 200, 600, 400)},
		"5":     sprite.SubTex{t, image.Rect(0, 400, 200, 600)},
		"6":     sprite.SubTex{t, image.Rect(200, 400, 400, 600)},
		"7":     sprite.SubTex{t, image.Rect(400, 400, 600, 600)},
		"8":     sprite.SubTex{t, image.Rect(0, 600, 200, 800)},
		"9":     sprite.SubTex{t, image.Rect(200, 600, 400, 800)},
		"GO":    sprite.SubTex{t, image.Rect(0, 800, 600, 1000)},
		"gooon": sprite.SubTex{t, image.Rect(0, 1000, 600, 1200)},
	}
}
