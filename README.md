# nSteg.
nSteg is a command-line utility, used to perform LSB steganography on images. It is written in <a href="https://golang.org/" target="_blank">Go</a> and make use of <a href="https://pkg.go.dev/github.com/auyer/steganography" target="_blank">auyer's Steganography lib</a> (also written in Go) for encoding/decoding purposes. At the time these lines are written, it's only supports `.png` images and `.txt` text files.

## Installation.
Step 1.
`cd` to `$GOPATH/src/github` directory of your Go installation and clone with the following command:
```
$ git clone https://github.com/niconc/nSteg.git
```
Step 2.
Make sure you have install the library mentioned above (it's a dependency), by using:
```
$ go get -u github.com/auyer/steganography
```

## Usage.
Into the installed directory (I'm assuming it's `$GOPATH/src/github.com/niconc/nSteg/`) create 2 additional directories: `images/` and `messages/` which will be used to store the images and text file messages respectively. The image and text files **must** be reside there.

**You have the following options:**

**Encode:** The process of encoding a text file as a message into an image:
```
./nSteg -coding=encode -image=images/someImage.png -text=messages/someText.txt
```
The process creates a new image file, with the same name as the original + `"_en"` + `.png` located at the `images` directory.

**Decode:** The process of decoding an already encoded image, `someImage_en.png`, extract the message, saves it to file with the same name as the original text, + `_en` + `.txt`. ******
```
./nSteg -coding=decode -image=images/someEncodedImage_en.png
```
---
_****Attention:** The `_en` extension **must** exist after the file name of the image (and before the `.png`) in order for the decoder to recognize the file as an encoded file._