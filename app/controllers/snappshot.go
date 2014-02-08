package controllers

import (
	"github.com/robfig/revel"
	"os"
	"strings"
	"strconv"
	"sort"
	"net/http"
	"time"

	"encoding/xml"
	"io/ioutil"
	"github.com/JustinJudd/go_snappshot/app/routes"

	"github.com/gographics/imagick/imagick"

)

// Location where all generated images will be saved
const GENERATED_DIR = "generated/"

func init() {
	os.Mkdir(GENERATED_DIR, os.ModeDir | os.ModePerm )
}

// Snappshot Controller
type Snappshot struct {
	*revel.Controller
}

// type representing image as can be returned to user in browser
type JPGImage []byte

// Set HTTP header types for returning image(not html page) 
func (r JPGImage) Apply(req *revel.Request, resp *revel.Response) {
	// Output screenshot
	resp.WriteHeader(http.StatusOK, "image/jpg")
	resp.Out.Write(r)
}

// Backgrounds contains a list of all potential background images
type Backgrounds struct {
	imageList []int
	imageMap  map[int]string
}

// The dimensions of a screenshot to be placed
type Dimensions struct {
	width  int
	height int
}

// The resolution of a background image
type Resolution struct {
	Width  string `xml:"width"`
	Height string `xml:"height"`
}

//The placement of a screenshot image onto a background image
type Placement struct {
	Top_left_x     string `xml:"top_left_x"`
	Top_left_y     string `xml:"top_left_y"`
	Top_right_x    string `xml:"top_right_x"`
	Top_right_y    string `xml:"top_right_y"`
	Bottom_left_x  string `xml:"bottom_left_x"`
	Bottom_left_y  string `xml:"bottom_left_y"`
	Bottom_right_x string `xml:"bottom_right_x"`
	Bottom_right_y string `xml:"bottom_right_y"`
}

// A device screenshot representation, how big it is and where it should go
type Device struct {
	Resolution Resolution `xml:"resolution"`
	Placement  Placement  `xml:"placement"`
}

// The overall resukting image information. The background image and all of the information for the screenshot image to be placed
type Result struct {
	XMLName  xml.Name `xml:"background"`
	Location string   `xml:"location"`
	Device   Device   `xml:"device"`
}



// Main page - show first background image
func (c Snappshot) Index() revel.Result {
	return c.Redirect(routes.Snappshot.Upload(1))
}

// Show FAQs page
func (c Snappshot) FAQ() revel.Result {
	return c.Render()
}

// Basic page showing a background and gives user option to upload screenshot
func (c Snappshot) Upload(image_id int) revel.Result {
	backgrounds := getImageRange(image_id)
	xml_data := loadBackgroundXML(strconv.Itoa(image_id))

	return c.Render(image_id, backgrounds, xml_data)
}

// User has uploaded screenshot - place onto background image
func (c Snappshot) Uploaded(image_id int, file_input []byte) revel.Result {
	backgrounds := getImageRange(image_id)
	image_str := strconv.Itoa(image_id)
	xml_data := loadBackgroundXML(image_str)
	placed_id := placeit(image_str, xml_data, file_input)
	return c.Render(image_id, backgrounds, placed_id)
}

func (c Snappshot) UploadedGet(image_id int) revel.Result {
	return c.Redirect(routes.Snappshot.Upload(image_id))
}

// For returning the image itself, read saved image from generated path
func (c Snappshot) Image(image_id string) revel.Result {

	data, err := ioutil.ReadFile(getGeneratedPath(image_id))
	if err != nil {
		c.Flash.Error("Unable to access image")
		c.Redirect(routes.Snappshot.Index())
	}
	return JPGImage(data)
}


func (c Snappshot) Screenshot(res string) revel.Result {
	s := strings.Split(res, "x")
	width, _ := strconv.Atoi(s[0])
	height, _ := strconv.Atoi(s[1])
	imagick.Initialize()
	defer imagick.Terminate()

	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	dw := imagick.NewDrawingWand()
	defer dw.Destroy()
	cw := imagick.NewPixelWand()
	cw2 := imagick.NewPixelWand()

	cw.SetColor("darkgray")
	cw2.SetColor("white")
	mw.NewImage(uint(width), uint(height), cw)

	dw.SetTextAlignment(imagick.ALIGN_CENTER)
	dw.SetFillColor(cw2)
	dw.SetFontSize(150)
	cw2.SetColor("black")
	dw.SetStrokeColor(cw2)
	dw.Annotation(float64(width)/2, float64(height)/2, res)

	mw.DrawImage(dw)
	mw.SetImageFormat("jpg")
	output := mw.GetImageBlob()
	return JPGImage(output)
}

// Ensure that a given string is an image name
func isImage(name string) bool {
	if strings.HasSuffix(name, ".jpg") {
		return true
	}
	return false
}

// Create Backgrounds by loading all background images and theur data
func getImageRange(place int) Backgrounds {
	imageplace := "src/github.com/JustinJudd/go_snappshot/public/images/backgrounds"
	imagedir, _ := os.Open(imageplace)
	files, _ := imagedir.Readdir(-1)
	var images []int
	var imagemap = make(map[int]string)
	for _, file := range files {
		if isImage(file.Name()) {

			thisplace, _ := strconv.Atoi(strings.TrimRight(file.Name(), ".jpg"))
			images = append(images, thisplace)
			if place == thisplace {
				imagemap[thisplace] = "active"
			} else {
				imagemap[thisplace] = ""
			}
		}
	}
	sort.Ints(images)
	return Backgrounds{images, imagemap}
}

// Loads XML data that contains info about where screenshots should be placed on background
func loadBackgroundXML(image_id string) Result {

	v := Result{}

	xml_path := "src/github.com/JustinJudd/go_snappshot/public/images/backgrounds/" + image_id + ".xml"
	data, err := ioutil.ReadFile(xml_path)
	if err != nil {
		println("Error reading background xml for ", image_id, ":", err.Error())
		return Result{}
	}

	err = xml.Unmarshal([]byte(data), &v)
	if err != nil {
		println("Error loading background xml for ", image_id, ":", err.Error(), v.Location)
		return Result{}
	}
	return v

}

// Actually placed screenshot onto background image
func placeit(image_id string, xml_data Result, image []byte) string {
	imagick.Initialize()
	defer imagick.Terminate()

	width, _ := strconv.Atoi(xml_data.Device.Resolution.Width)
	height, _ := strconv.Atoi(xml_data.Device.Resolution.Height)

	x_positions := make([]int, 4)
	x_positions[0], _ = strconv.Atoi(xml_data.Device.Placement.Top_left_x)
	x_positions[1], _ = strconv.Atoi(xml_data.Device.Placement.Top_right_x)
	x_positions[2], _ = strconv.Atoi(xml_data.Device.Placement.Bottom_right_x)
	x_positions[3], _ = strconv.Atoi(xml_data.Device.Placement.Bottom_left_x)

	y_positions := make([]int, 4)
	y_positions[0], _ = strconv.Atoi(xml_data.Device.Placement.Top_left_y)
	y_positions[1], _ = strconv.Atoi(xml_data.Device.Placement.Top_right_y)
	y_positions[2], _ = strconv.Atoi(xml_data.Device.Placement.Bottom_right_y)
	y_positions[3], _ = strconv.Atoi(xml_data.Device.Placement.Bottom_left_y)

	base_x := x_positions[0]
	base_y := y_positions[0]

	result := []float64{0, 0,
		float64(x_positions[0] - base_x), float64(y_positions[0] - base_y),
		float64(width), 0,
		float64(x_positions[1] - base_x), float64(y_positions[1] - base_y),
		float64(width), float64(height),
		float64(x_positions[2] - base_x), float64(y_positions[2] - base_y),
		0, float64(height),
		float64(x_positions[3] - base_x), float64(y_positions[3] - base_y)}


	mw := imagick.NewMagickWand()
	defer mw.Destroy()
	back_mw := imagick.NewMagickWand()
	defer back_mw.Destroy()

	mw.ReadImageBlob(image)
	back_mw.ReadImage("src/github.com/JustinJudd/go_snappshot/public/images/backgrounds/" + image_id + ".jpg")

	mw.SetImageVirtualPixelMethod(imagick.VIRTUAL_PIXEL_TRANSPARENT)
	mw.DistortImage(imagick.DISTORTION_PERSPECTIVE, result, true)

	sort.IntSlice.Sort(x_positions)
	sort.IntSlice.Sort(y_positions)
	back_mw.CompositeImage(mw, imagick.COMPOSITE_OP_OVER, x_positions[0], y_positions[0])
	placed_id := strconv.Itoa(int(time.Now().UnixNano()))
	back_mw.WriteImage(getGeneratedPath(placed_id))

	return placed_id

}

// Get the location of a generated image
func getGeneratedPath(image_id string) string {
	return GENERATED_DIR + image_id + ".jpg"
}

// Get a list of all backgrounds
func (b Backgrounds) GetImageList() []int {
	return b.imageList
}

// Get a mapping of all backgrounds
func (b Backgrounds) GetImageMap() map[int]string {
	return b.imageMap
}

// Get the active background(which one is the main image)
func (b Backgrounds) GetActive(image int) string {
	return b.imageMap[image]
}
