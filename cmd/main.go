package main

import (
	"log"
	"time"

	"github.com/ffx64/discord-rpc-go/client"
	"github.com/google/uuid"
)

func main() {
	// Create a new Discord RPC client with your application's ID
	cli := client.NewClient("your_application_id_here")
	cli.SetVerbose(true)

	// Event called when the connection to Discord is ready
	cli.OnReady(func(info map[string]any) {
		log.Println("READY event, info:", info)
	})

	// Event called when an error occurs
	cli.OnError(func(err error) {
		log.Println("Error:", err)
	})

	// Event called when someone joins your activity via Rich Presence
	cli.OnActivityJoin(func(secret string) {
		log.Println("ACTIVITY_JOIN secret:", secret)
	})

	// Connect to Discord IPC
	if err := cli.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer cli.Close()

	// Define an activity to display in Discord Rich Presence
	activity := client.Activity{
		Type:    client.Playing,
		State:   "Editing a project",
		Details: "Working on Go modules",
		Assets: &client.Assets{
			LargeImage: "ide-logo",
			SmallImage: "editor-icon",
			LargeText:  "My IDE",
			SmallText:  "Code Editor",
		},
		Timestamps: &client.Timestamps{
			Start: time.Now().Unix(),
		},
		Party: &client.Party{
			ID:   uuid.New().String(),
			Size: []int{1, 5}, // current, max
		},
		Buttons: []client.Button{
			{Label: "Open Project Docs", Url: "https://example.com/docs"},
			{Label: "Visit Repository", Url: "https://example.com/repo"},
		},
	}

	// Set the activity in Discord
	if err := cli.SetActivity(activity); err != nil {
		log.Fatalf("Failed to set activity: %v", err)
	}

	log.Println("Activity set successfully, keeping the process alive...")
	select {} // keep the program running
}
