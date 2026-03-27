package main

import (
	_ "embed"
	"log"
	"time"

	"fyne.io/systray"
	"github.com/liamchampton/copilot-cli-session-monitor/internal/menu"
	"github.com/liamchampton/copilot-cli-session-monitor/internal/monitor"
	"github.com/liamchampton/copilot-cli-session-monitor/internal/session"
)

//go:embed assets/icon-active.png
var iconActive []byte

//go:embed assets/icon-idle.png
var iconIdle []byte

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	reader, err := session.NewReader("")
	if err != nil {
		log.Fatalf("Failed to initialize session reader: %v", err)
	}

	builder := menu.NewBuilder(iconActive, iconIdle)
	mon := monitor.New(reader, builder, 30*time.Second)
	quitCh := mon.Start()

	go func() {
		<-quitCh
		mon.Stop()
		systray.Quit()
	}()
}

func onExit() {}
