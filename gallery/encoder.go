package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ImageMeta struct {
	Filename    string `json:"filename"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

func main() {
	outDir := flag.String("output", "./", "Output directory for encoded files")
	flag.StringVar(outDir, "o", "./", "Output directory for encoded files (shorthand)")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Println("Usage: rattz-xyz /path/to/images metadata.json [-o output-dir/]")
		os.Exit(1)
	}

	imagesPath := args[0]
	metaFile := args[1]

	metaBytes, err := os.ReadFile(metaFile)
	if err != nil {
		panic(err)
	}

	var metas []ImageMeta
	if err := json.Unmarshal(metaBytes, &metas); err != nil {
		panic(err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		panic(err)
	}

	for _, meta := range metas {
		imgPath := filepath.Join(imagesPath, meta.Filename)
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			panic(fmt.Sprintf("Failed to read image %s: %v", imgPath, err))
		}

		outFile := filepath.Join(*outDir, strings.TrimSuffix(meta.Filename, filepath.Ext(meta.Filename))+".rio")
		out, err := os.Create(outFile)
		if err != nil {
			panic(err)
		}

		mBytes, _ := json.Marshal(meta)
		binary.Write(out, binary.LittleEndian, uint32(len(mBytes)))
		_, _ = out.Write(mBytes)

		binary.Write(out, binary.LittleEndian, uint32(len(imgData)))
		_, _ = out.Write(imgData)
		out.Close()

		fmt.Printf("Encoded: %s -> %s\n", meta.Filename, outFile)
	}

	fmt.Println("All images encoded.")
}
