# Discord RPC Go Client

A Go library for integrating with Discord Rich Presence via IPC, allowing custom activities to be displayed on the user's profile, including details, status, images, and interactive buttons.

---

## Installation

```bash
go get github.com/ffx64/discord-rpc-go/client
```

---

## Key Concepts

### `Client`

The main client that connects to Discord IPC and sends activities (`Activity`).

**Main methods:**

- `NewClient(appID string) *Client` – Creates a new client instance with your Discord application ID.
- `Connect() error` – Connects to Discord IPC.
- `Close()` – Closes the connection.
- `SetActivity(act Activity) error` – Sets the current user activity.
- `OnReady(func(info map[string]any))` – Callback triggered when the connection is ready.
- `OnError(func(err error))` – Callback for errors.
- `OnActivityJoin(func(secret string))` – Callback when someone joins your session via Rich Presence.

---

### `Activity`

Structure that defines the activity to be displayed on the user's profile.

```go
type Activity struct {
	Type       ActivityType
	State      string
	Details    string
	Timestamps *Timestamps
	Assets     *Assets
	Party      *Party
	Secrets    map[string]string
	Buttons    []Button
}
```

#### Important fields:

- `Type` – Activity type (`Playing`, `Listening`, `Watching`, `Competing`). `Streaming` is not accepted via official IPC.
- `State` – Bottom line text of the activity.
- `Details` – Top line text of the activity.
- `Timestamps` – Activity start and end time.
- `Assets` – Large and small images with optional descriptive texts and clickable URLs.
- `Party` – Information about the user's party (`ID` and current/max size).
- `Secrets` – For join, spectate, or match secret.
- `Buttons` – Clickable buttons (max 2), each with a `Label` and `URL`.

---

### `ActivityType`

```go
type ActivityType int

const (
    Playing ActivityType = 0
    Listening ActivityType = 2
    Watching ActivityType = 3
    Competing ActivityType = 5
)
```

---

### `Timestamps`

```go
type Timestamps struct {
    Start int64 // Unix timestamp
    End   int64
}
```

---

### `Assets`

```go
type Assets struct {
    LargeImage string // large asset key
    LargeText  string // tooltip for large asset
    SmallImage string
    SmallText  string
}
```

---

### `Party`

```go
type Party struct {
    ID   string // unique party identifier
    Size []int  // [current, max]
}
```

> The `ID` is automatically generated if not provided. It is used by Discord to track members in the same game or group activity.

---

### `Button`

```go
type Button struct {
    Label string
    Url   string
}
```

- Maximum of two buttons.
- URLs must start with `http://`, `https://` or `steam://` (to launch games via Steam).

---

## Example Usage

```go
package main

import (
    "log"
    "time"

    "github.com/ffx64/discord-rpc-go/client"
    "github.com/google/uuid"
)

func main() {
    cli := client.NewClient("123456789012345678") // Your Discord app ID
    cli.SetVerbose(true)

    cli.OnReady(func(info map[string]any) {
        log.Println("READY event:", info)
    })

    cli.OnError(func(err error) {
        log.Println("Error:", err)
    })

    if err := cli.Connect(); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer cli.Close()

    act := client.Activity{
        Type:    client.Playing,
        State:   "Working on an important task",
        Details: "Status: Processing data",
        Assets: &client.Assets{
            LargeImage: "large-icon",
            LargeText:  "Productivity Tool",
            SmallImage: "small-icon",
            SmallText:  "Module active",
        },
        Timestamps: &client.Timestamps{
            Start: time.Now().Unix(),
        },
        Party: &client.Party{
            ID:   uuid.New().String(),
            Size: []int{3, 10},
        },
        Buttons: []client.Button{
            {Label: "Visit Website", Url: "https://example.com"},
            {Label: "Documentation", Url: "https://example.com/docs"},
        },
    }

    if err := cli.SetActivity(act); err != nil {
        log.Fatalf("Failed to set activity: %v", err)
    }

    log.Println("Activity set, keeping process alive...")
    select {}
}
```

---

### Important Notes

1. `Streaming` is not officially supported via Discord IPC.
2. Clickable buttons can open external URLs.
3. Party IDs must be unique and can be generated automatically.
4. Assets can include tooltips displayed on hover.
