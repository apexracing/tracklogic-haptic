package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simagic/hpr"
)

func main() {
	channel := flag.Int("ch", int(hpr.TargetBrake), "Channel: 0=Clutch, 1=Brake, 2=Throttle")
	freq := flag.Int("f", hpr.MaxFrequency, "Frequency 0-50; larger values are clamped")
	amp := flag.Int("a", hpr.MaxAmplitude, "Amplitude 0-100")
	duration := flag.Duration("d", 2*time.Second, "Duration (e.g. 3s, 500ms); 0 waits for Ctrl+C")
	list := flag.Bool("list", false, "List connected Simagic pedals only")
	flag.Parse()

	manager := hpr.NewManager(hpr.WithDefaultDrivers())

	if *list {
		pedals, err := manager.Scan()
		if err != nil {
			log.Fatalf("Failed to scan: %v", err)
		}
		if len(pedals) == 0 {
			fmt.Println("No Simagic pedals detected.")
			return
		}
		fmt.Println("Detected Simagic pedals:")
		for i, p := range pedals {
			fmt.Printf("  [%d] %s\n    Path: %s\n", i, p.Model, p.DevicePath)
		}
		return
	}

	ch, err := parseChannel(*channel)
	if err != nil {
		log.Fatal(err)
	}
	f, err := parseFrequency(*freq)
	if err != nil {
		log.Fatal(err)
	}
	a, err := parseAmplitude(*amp)
	if err != nil {
		log.Fatal(err)
	}

	pedals, err := manager.Scan()
	if err != nil {
		log.Fatalf("Failed to scan pedals: %v", err)
	}
	if len(pedals) == 0 {
		log.Fatal("No Simagic pedals found. Make sure pedals are connected via USB.")
	}

	fmt.Printf("Found %d Simagic pedal(s):\n", len(pedals))
	for _, p := range pedals {
		fmt.Printf("  - %s\n", p.Model)
	}

	info := pedals[0]
	fmt.Printf("\nOpening %s ...\n", info.Model)
	dev, err := manager.Open(info)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	defer func() {
		if err := dev.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close device cleanly: %v\n", err)
			return
		}
		fmt.Println("Device closed.")
	}()

	fmt.Printf("Sending vibration to %s: frequency=%.0f, amplitude=%.0f, duration=%v\n",
		ch, f, a, *duration)

	if err := dev.Vibrate(hpr.Command{
		Target:    ch,
		State:     hpr.On,
		Frequency: f,
		Amplitude: a,
	}); err != nil {
		log.Fatalf("Failed to send vibration: %v", err)
	}

	if *duration > 0 {
		fmt.Printf("Vibrating for %v ...\n", *duration)
		time.Sleep(*duration)
		fmt.Println("Stopping.")
		if err := dev.Stop(ch); err != nil {
			log.Fatalf("Failed to stop vibration: %v", err)
		}
		return
	}

	fmt.Println("Vibrating until Ctrl+C ...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\nStopping.")
	if err := dev.Stop(ch); err != nil {
		log.Fatalf("Failed to stop vibration: %v", err)
	}
}

func parseChannel(v int) (hpr.Target, error) {
	ch := hpr.Target(v)
	if !ch.Valid() {
		return 0, fmt.Errorf("invalid channel: %d (must be 0, 1, or 2)", v)
	}
	return ch, nil
}

func parseFrequency(v int) (float32, error) {
	if v < hpr.MinFrequency {
		return 0, fmt.Errorf("frequency must be >= %d", hpr.MinFrequency)
	}
	if v > hpr.MaxFrequency {
		return float32(hpr.MaxFrequency), nil
	}
	return float32(v), nil
}

func parseAmplitude(v int) (float32, error) {
	if v < hpr.MinAmplitude || v > hpr.MaxAmplitude {
		return 0, fmt.Errorf("amplitude must be %d-%d", hpr.MinAmplitude, hpr.MaxAmplitude)
	}
	return float32(v), nil
}
