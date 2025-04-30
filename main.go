package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	_ "github.com/VincentApta/cctv-timbangan/docs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title EZVIZ RTSP Snapshot API
// @version 1.0
// @description Capture snapshot from EZVIZ camera over RTSP.
// @host localhost:3000
// @BasePath /

// Snapshot model
type Snapshot struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Filename   string    `json:"filename"`
	CapturedAt time.Time `json:"captured_at"`
}

// Request payload
type CaptureRequest struct {
	IP   string `json:"ip" example:"192.168.68.126"`
	Code string `json:"code" example:"QZNWZI"`
}

var db *gorm.DB

func initDB() {
	dsn := "host=localhost user=postgres password=postgres dbname=ezviz port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Snapshot{})
}

// CaptureSnapshot runs ffmpeg to save a frame
func CaptureSnapshot(rtspURL, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", rtspURL, "-frames:v", "1", "-q:v", "2", outputPath)
	return cmd.Run()
}

func CaptureMultipleSnapshots(ip, code string, count int) []Snapshot {
	timestamp := time.Now().UnixNano()
	outputPattern := fmt.Sprintf("snapshots/snapshot_%d_%%d.jpg", timestamp)
	rtspURL := fmt.Sprintf("rtsp://admin:%s@%s:554/h264", code, ip)

	cmd := exec.Command("ffmpeg", "-rtsp_transport", "tcp", "-y", "-i", rtspURL, "-vf", "scale=640:-1", "-frames:v", fmt.Sprint(count), "-q:v", "2", outputPattern)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Failed to capture multiple:", err)
		return nil
	}

	// Save snapshot metadata
	var results []Snapshot
	for i := 1; i <= count; i++ {
		filename := fmt.Sprintf("snapshot_%d_%d.jpg", timestamp, i)
		fmt.Println("Captured:", filename)
		snap := Snapshot{Filename: filename, CapturedAt: time.Now()}
		db.Create(&snap)
		results = append(results, snap)
	}

	return results
}

// CaptureMultipleHandler godoc
// @Summary Capture multiple snapshots
// @Accept json
// @Produce json
// @Param data body CaptureRequest true "Camera IP and RTSP code"
// @Success 200 {array} Snapshot
// @Router /capture-multiple [post]
func CaptureMultipleHandler(c *fiber.Ctx) error {
	var req CaptureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).SendString("Invalid input")
	}

	snaps := CaptureMultipleSnapshots(req.IP, req.Code, 5)
	return c.JSON(snaps)
}

func main() {
	app := fiber.New()

	// Enable CORS for Swagger UI and browser clients
	app.Use(cors.New())

	initDB()

	dir, _ := os.Getwd()
	app.Static("/snapshots", dir+"/snapshots")

	app.Post("/capture-multiple", CaptureMultipleHandler)

	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Listen(":3000")
}
