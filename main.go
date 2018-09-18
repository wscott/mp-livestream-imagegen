// mp-livestream-imagegen
//
// This program is used to generate graphic overlays for a video feed
// from our church. Mostly lower third boxes with a logo and some
// formatted test.
//
// It reads a TOML formatted spec file into the following structures
// and then processes them.  The fields in the file match the
// structure names.

package main

import "fmt"
import "os"
import "github.com/fogleman/gg"
import "github.com/BurntSushi/toml"
import "github.com/docopt/docopt-go"

// Information about the current speaker. Used to create a name badge. NYI
type speaker struct {
	Name string
	Role string
}

// Toplevel config
type setup struct {
	Width, Height int    // dimentions of final image
	Size          int    // height of lower third box
	LogoWidth     int    `toml:"logo_width"`
	Icon          string // path to logo image
	Background    string // image to tile for background
	Font          string // path to TrueType font file
	VerseSize     int    `toml:"verse_size"` // font size of verse
	TitleSize     int    `toml:"title_size"` // font size of titles
	Border        int    // space to edge of image
}

// information about an individual slide
type slide struct {
	// for sermon points
	Title    string // main point
	Subtitle string // subpoint

	// for verses or larger text blocks
	Text string // block of text
	Ref  string // bible reference at bottom-right

	// for image slides
	Image string // path to image to put on right of screen
}

type fData struct {
	Setup   setup
	Speaker speaker
	Slide   []slide
}

// generate the background for a lower-third slide.  A filled
// rectangle is put at the bottom with a logo centered on the left side.
func defaultSlide(fdata fData, dc *gg.Context) {
	dc.DrawRectangle(0, float64(fdata.Setup.Height-fdata.Setup.Size),
		float64(fdata.Setup.Width), float64(fdata.Setup.Size))
	img, err := gg.LoadImage(fdata.Setup.Background)
	if err != nil {
		panic(err)
	}
	dc.SetFillStyle(gg.NewSurfacePattern(img, gg.RepeatBoth))
	dc.Fill()
	scaleImageToBox(dc, fdata.Setup.Icon,
		fdata.Setup.Border,
		fdata.Setup.Height-fdata.Setup.Size+fdata.Setup.Border,
		fdata.Setup.LogoWidth-fdata.Setup.Border,
		fdata.Setup.Size-2*fdata.Setup.Border,
		0.5, 0.5)
}

// Generate a verse slide. Just a wrapped block of text with a
// reference at the bottom. This code does check if the text actually
// fit, it will just keep going off the bottom of the slide. The uses
// needs to notice this and split the slide if needed.
func doVerse(fdata fData, s slide, dc *gg.Context) {
	err := dc.LoadFontFace(fdata.Setup.Font, float64(fdata.Setup.VerseSize))
	if err != nil {
		panic(err)
	}
	dc.SetRGB(0, 0, 0)
	dc.DrawStringWrapped(s.Text,
		float64(fdata.Setup.LogoWidth+fdata.Setup.Border),
		float64(fdata.Setup.Height-fdata.Setup.Size+
			fdata.Setup.Border),
		0, 0,
		float64(fdata.Setup.Width-fdata.Setup.LogoWidth-
			2*fdata.Setup.Border),
		1.5, gg.AlignLeft)
	if len(s.Ref) > 0 {
		dc.DrawStringAnchored(s.Ref,
			float64(fdata.Setup.Width-fdata.Setup.Border),
			float64(fdata.Setup.Height-fdata.Setup.Border),
			1, 0)
	}
}

// Generate main point slide.  Text centered in a larger font.
func doMainPoint(fdata fData, s slide, dc *gg.Context) {
	err := dc.LoadFontFace(fdata.Setup.Font, float64(fdata.Setup.TitleSize))
	if err != nil {
		panic(err)
	}
	dc.SetRGB(0, 0, 0)
	dc.DrawStringAnchored(s.Title,
		float64(fdata.Setup.Size+
			(fdata.Setup.Width-fdata.Setup.LogoWidth)/2),
		float64(fdata.Setup.Height-fdata.Setup.Size/2),
		.5, .5)
}

func scaleImageToBox(dc *gg.Context, file string,
	x, y, w, h int, ax, ay float64) {

	img, err := gg.LoadImage(file)
	if err != nil {
		panic(err)
	}
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()

	// How much do I need to scale the image to fit vertically and
	// horizontally.
	scalex := float64(w) / float64(iw)
	scaley := float64(h) / float64(ih)
	scale := scalex
	if scale > scaley {
		scale = scaley
	}

	// Draw scaled image. The position is also scaled so I needed
	// to reverse scale the y position.
	dc.Scale(scale, scale)
	dc.DrawImage(img,
		int(float64(x)/scale+(float64(w)/scale-float64(iw))*ax),
		int(float64(y)/scale+(float64(h)/scale-float64(ih))*ay))
	dc.Identity()
}

// Generate image slide.  The image is scaled to fit on the left side
// of the screen. It is also moved up so a lower third image can be
// added without overlap.
func doImage(fdata fData, s slide, dc *gg.Context) {

	scaleImageToBox(dc, s.Image, 0, 0,
		fdata.Setup.Width/2, fdata.Setup.Height-fdata.Setup.Size,
		0, 0.5)
}

func doSlide(fdata fData, s slide, name string) {
	dc := gg.NewContext(fdata.Setup.Width, fdata.Setup.Height)
	if len(s.Image) == 0 {
		defaultSlide(fdata, dc)
	}
	if len(s.Ref) > 0 {
		doVerse(fdata, s, dc)
	} else if len(s.Title) > 0 {
		doMainPoint(fdata, s, dc)
	} else if len(s.Image) > 0 {
		doImage(fdata, s, dc)
	}
	dc.SavePNG(name)
}

func main() {
	usage := `Livestream image generator

Usage:
  mp-livestream-imagegen SPECFILE

Options
  -h --help   Show this screen.
  --version   Show version.`

	args, _ := docopt.ParseDoc(usage)

	var fdata fData
	_, err := toml.DecodeFile(args["SPECFILE"].(string), &fdata)
	if err != nil {
		fmt.Println(err)
		return
	}
	if fdata.Setup.LogoWidth == 0 {
		fdata.Setup.LogoWidth = fdata.Setup.Size
	}
	os.Mkdir("output", 0777)
	for cnt, s := range fdata.Slide {
		doSlide(fdata, s,
			fmt.Sprintf("output/slide%02d.png", cnt+1))
	}
}
