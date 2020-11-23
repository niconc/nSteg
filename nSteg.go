package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/dustin/go-humanize"

	/* Import the _ packages purely for their initialization side effects. That means if we want to decode an image, the image types like PNG, JPG, GIF must be registered so the function image.Decode to understand its format. We're achieveing this behaviur by importing them with an underscore _ */

	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"log"
	"os"

	"github.com/auyer/steganography"
)

var (
	decMess        []byte        // the decoded message from image.
	messNameToSave string        // the name of the file to save the extracted message.
	flagImage      string        // flag to hold the image name as a flag parameter.
	flagText       string        // flag to hold the image name as a flag parameter.
	flagEncDec     string        // flag to hold the encoding or decoding command.
	fileInfo       os.FileInfo   // file information. THE FILE INFO NOW IS ON SANITY CHECK. THIS MUST BE FOR EVERY FILE SEPARATELY.
	sizeOfMessage  uint32        // the size of a message the image can store.
	imgFile        *os.File      // represents an Open image file for read.
	imgReader      *bufio.Reader // reader wrapper for image file reader.
	imgDecoded     image.Image   // decoding image to image.Image interface.
	imgType        string        // the type of ORIGINAL image (PNG, JPEG, GIF)
	imgNameToSave  string        // the name of the encoded image file that will be saved.
	imgEncodedTxt  *os.File      // the final image with text encoded.
	txtFile        *os.File      // represents an Open text file for read.
	txtBS          []byte        // holds the bytes of the readed text file.
	err            error         // error var.
)

func main() {
	fmt.Println("A Command-line Steganography Utility.")

	// Precheck:
	/* Flags and Arguments ************************************************************
	**                     ************************************************************	/
	/* Setup Flags:	[SAMPLE Encoding]->
	$ go run nSteg.go -encode -image=filename.gif|jpg|png -text=filename.txt	*/
	/* Setup Flags:	[SAMPLE Decoding]->
	$ go run nSteg.go -decode -image=filename.gif|jpg|png	*/
	precheck()      // run check for basic flags/arguments errors.
	flagsArgsTest() // parse the command-line and perform tests.

	// Begin Process:
	/* ENCODE Message to an image ********************************************************
	* ******************************************************************************** */
	if flagEncDec == "encode" {
		openImage()
		defer imgFile.Close() // defering the close() file
		openText()
		defer txtFile.Close() // defering the close() file

		/* Perform sanity check by reading some bytes from the begining of the file and perform other tests. */
		sanityCheck(imgFile, 5, "Image")
		sanityCheck(txtFile, 5, "Text")
		fmt.Printf("\n")

		/* Create bufio.Reader objects by wraping the existing file readers above (*File), and with the use of buffer, to achieve better perfomance. */
		/* Text File: */
		objectText()

		/* Image File: Create a new file name: construct path/name and type to save */
		objectImage()

		/* Final stage: encode text to image and save. */
		encodeTextToImage()
	}

	/* DECODE Message from image *********************************************************
	* ******************************************************************************** */
	if flagEncDec == "decode" {
		openImage()
		defer imgFile.Close() // defering the close() file

		/* Create a new file name: construct path/name and type to save. */
		decMessage()
		messNameToSave = constructFileName(imgFile.Name(), "text")
		saveDecMessageToFile(messNameToSave) // save to file
		defer txtFile.Close()                // defering the close() file
	}
}

func saveDecMessageToFile(fileName string) {
	// Create the file.
	txtFile, err = os.Create(fileName)
	checkError("Error creating text file for writing message: ", err)

	// Create a buffer and save to it.
	w := new(bytes.Buffer)     // a new buffer
	n, err := w.Write(decMess) // write (append) []byte to w, from decMess (decMessage()).
	fmt.Printf("\nNumber of bytes readed during decoding: %d\n", n)
	checkError("Error in writing text from message, in buffer: ", err)

	// read to string from w and print (BEFORE THE BUFFER DUMPED TO FILE AND BE EMPTY!!!).
	message := w.String()
	fmt.Println(message)

	// write []byte to file and print the message.
	n2, err := w.WriteTo(txtFile)
	fmt.Printf("\nNumber of bytes written to file: %d\n", n2)
	checkError("Error in writing text from message, in buffer: ", err)

}

func decMessage() {
	var mes string
	mes = imgFile.Name()
	fmt.Println(mes)

	// Check if the name contains "_en" before the extension.
	if enc := strings.Contains(mes, "_en"); !enc {
		log.Fatalf("\nThe file to be decoded MUST have \"_en\" before the file extension.\nexample: %v\n", "\"messages/filename_en.png\"")
	}
	// Create a reader
	imgReader = bufio.NewReader(imgFile)    // buffer reader
	imgDecoded, err = png.Decode(imgReader) // decoding to golang's image.Image
	// retrieving message size to decode in the next line
	sizeOfMessage = steganography.GetMessageSizeFromImage(imgDecoded)
	// decoding the message from file.
	decMess = steganography.Decode(sizeOfMessage, imgDecoded)
}

func encodeTextToImage() {
	/* Decoding ORIGINAL image and information */
	fmt.Printf("::Encoding & Saving::\n")
	imgDecoded, imgType, err = imgFileDataDecoder(imgFile)

	/* Encoding text to image: */
	w := new(bytes.Buffer)
	err = steganography.Encode(w, imgDecoded, txtBS)
	checkError("Error encoding text in image: ", err)

	imgEncodedTxt, err = os.Create(imgNameToSave)
	checkError("Error creating new file: ", err)
	n, err := w.WriteTo(imgEncodedTxt) // write the *bytes.Buffer to file.
	fmt.Printf("\nNumber of bytes written to file during encoding: %s bytes (~ %s)\n",
		humanize.Comma(int64(n)), humanize.Bytes(uint64(n)))
	checkError("Error writing to new file: ", err)
}

func objectImage() {
	/* Create a new file name: construct path/name and type to save. */
	imgNameToSave = constructFileName(imgFile.Name(), "image")
	fmt.Println("Encoded file name to be saved:")
	fmt.Println(imgNameToSave)
	fmt.Printf("\n")
}

func objectText() {
	/* Text File: call func and convert reader data to []byte, OR do the same with one-liner: txtBS = ioutil.ReadAll(txtFile) */
	fmt.Printf("::Text File::\n")
	txtBS = txtFileRead(txtFile)
	fmt.Printf("Number of Unicode characters: %d\n", utf8.RuneCount(txtBS))
	fmt.Printf("\n\"\"\n")
	fmt.Printf(string(txtBS)) // print the result of readed text data.
	fmt.Printf("\n\"\"")
	fmt.Printf("\n\n")
}

func openImage() {
	/* Open the named image file and creating a reader for reading. */
	imgFile, err = os.Open(flagImage)                // the image file.
	checkError("Error in reading image file: ", err) // error checking.
}

func openText() {
	/* Open the named text file and creating a reader for reading. */
	txtFile, err = os.Open(flagText)                // the text file.
	checkError("Error in reading text file: ", err) // error checking.
}

func sanityCheck(file *os.File, nBytes int, tp string) {
	/* About file: */
	fmt.Printf("\n::Sanity Check::\n")
	fmt.Printf("Checking %s:\n", tp)
	fileInfo, err = file.Stat() // returns a info describing the file, and some methods.
	checkError("Error geting file statistics: ", err)

	// check if the file is empty.
	if fileInfo.Size() == 0 {
		fmt.Printf("\nError reading file %q. The file is empty. Please select another file.\n", fileInfo.Name())
		os.Exit(1)
	}
	// check if file is a directory.
	if fileInfo.IsDir() == true {
		fmt.Printf("\nImproper file name %q. The file name is a directory. Please select another file.\n", fileInfo.Name())
		os.Exit(1)
	}
	// Read 5 bytes from file.,
	bytesTest := make([]byte, nBytes) // create a slice of n bytes.
	n, err := file.Read(bytesTest)    // read n bytes from file.
	checkError("Error reading bytes from file, during sanity check: ", err)
	fmt.Printf("Bytes from file %q : %d bytes: %d string: %s\n", file.Name(), n, bytesTest[:n], string(bytesTest[:n]))
	file.Seek(0, 0) // return the offest for the next read/write to 0.
}

func precheck() {
	switch {
	case len(os.Args) == 1:
		fmt.Println("Flags/parameters are missing.")
		usage()
	case len(os.Args) > 4:
		fmt.Println("Too many flags/parameters. Max number is 3.")
		usage()
	case len(os.Args) < 3:
		fmt.Println("Too few flags/parameters. Min number is 2.")
		usage()
	}
}

func flagsArgsTest() {
	/* We're using plain pre-declared vars and generate pointers inside the func. */
	flag.StringVar(&flagEncDec, "coding", "none",
		"Encoding / Decoding selection of process.") // 1st flag.
	flag.StringVar(&flagImage, "image", "none",
		"Image file name, along with the subpath.") // 2nd flag.
	flag.StringVar(&flagText, "text", "none",
		"Text file name, along with the subpath") // 3rd flag.

	/* Parse the flags and assign the values to the variables */
	flag.Parse()

	/* Test flags and arguments:
	if "none" -> no flags at all are passed after the command,
	if "" -> the flags are passed without any values. e.g. -image= */
	switch {
	// check Encode/Decode flag presence & value.
	case flagEncDec == "none" || flagEncDec == "":
		fmt.Printf("\nPlease select process of encoding/decoding\nOR check the flags/names or spelling errors.\n")
		usage() // call usage() to display help and exit.

		// If encode...
	case flagEncDec == "encode":
		if len(os.Args) < 4 {
			fmt.Println("Not enough arguments for encoding.")
			usage() // call usage() to display help and exit.
		}
		switch {
		case (flagImage == "none" && flagText == "none") ||
			(flagImage == "" && flagText == ""):
			fmt.Printf("\nPlease provide a valid path/filename for both the image and the text file,\nOR check the flags/names for spelling errors.\n")
			usage() // call usage() to display help and exit.
		case flagImage == "none" || flagImage == "":
			fmt.Printf("\nPlease provide a valid path/filename for the image file,\nOR check the flags/names for spelling errors.\n")
			usage() // call usage() to display help and exit.
		case flagText == "none" || flagText == "":
			fmt.Print("\nPlease provide a valid path/filename for the text file,\nOR check the flags/names for spelling errors.\n")
			usage() // call usage() to display help and exit.
		}

		// If decode...
	case flagEncDec == "decode":
		if len(os.Args) > 3 {
			fmt.Println("Too many arguments for decoding.")
			usage() // call usage() to display help and exit.
		}
		switch {
		case flagImage == "none" || flagImage == "":
			fmt.Printf("\nPlease provide a valid path/filename for the image file,\nOR check the flags/names for spelling errors.\n")
			usage() // call usage() to display help and exit.
		}
	}

	/* Flags and arguments from os.Args point of view */
	fmt.Printf("\n::Flags and arguments as parameters (os package pov)::\n")
	fmt.Println(os.Args)
	for i := 0; i < len(os.Args); i++ {
		fmt.Printf("os.Args num. %v: %v\n", i, os.Args[i])
	}
	/* Flags and arguments from flags package point of view */
	fmt.Printf("\n::Flags and arguments as values (flags package pov)::\n")
	fmt.Printf("1st flag value:\t\t%v\n", flagEncDec)
	fmt.Printf("2nd flag value:\t\t%v\n", flagImage)
	fmt.Printf("3rd flag value:\t\t%v\n", flagText)
	fmt.Printf("Remaining, positional flag arguments:\t%v\n", flag.Args())
}

func usage() {
	fmt.Println(`
__________________________________
nSteg command-line utility usage:
[SAMPLE Encoding]->
$ ./nSteg encode -image=filename.gif|jpg|png -text=filename.txt

[SAMPLE Decoding]->
$ ./nSteg.go decode -image=filename.gif|jpg|png	*/

[Legend]->
 <  nSteg  > : runs the program
 < -coding=encode|decode  > : The first command :
		-encode To encode text into image, or
		-decode To decode text from image.
 < -image=someImage.jpg|png|gif > : Image flag :
    the image file to be encoded|decoded. It must be in .jpg|.png|.gif format.
 < -text=someFile.txt > : Text Flag : 
    the text file to be encoded into image in .txt format.
    When decoding, the text file produced, has always the same name as the image
    and it's always in in UTF8 .txt format.
	
`)
	os.Exit(1) // exit here, after the usage().
}

func txtFileRead(txtFile *os.File) []byte {
	var (
		bs        []byte        // the byte slice with the text we'll return.
		bufVal    []byte        // the buffer's byte slice, filled with every iteration.
		err       error         // errors.
		txtReader *bufio.Reader // reader wrapper for text *file reader.
	)
	/* Create bufio.Reader object by wraping the existing file reader (*File).
	Then, read from file */
	txtReader = bufio.NewReader(txtFile)

	/* Read from file */
	for {
		bufVal, err = txtReader.ReadBytes('\n')
		switch {
		case err == io.EOF:
			bs = append(bs, bufVal...)
			return bs
		case err != nil:
			checkError("Error reading from buffered file: ", err)
		}
		bs = append(bs, bufVal...)
	}
	return bs
}

func constructFileName(filePathName string, tp string) string {
	/* This function works for constructing the encoded image name
	   but ALSO for the extracted message from an image. */
	var (
		path      string // the path of the original image.
		name      string // the name of the original image.
		ext       string // the extension of the original image.
		finalName string // the new path AND name to use for saving.
	)
	path, name = filepath.Split(filePathName) // get dir and name of the original image.
	ext = filepath.Ext(filePathName)          // get the OLD extension from file. (GETS THE .png FOR EXAMPLE. IT COULD BE RETURNED FROM HERE ->)
	name = strings.TrimSuffix(name, ext)      // remove the OLD file name extension.

	switch tp {
	// if type is image
	case "image":
		finalName = path + name + "_en" + ext // construct the final IMAGE name.
	// if type is text
	case "text":
		path = "messages/" // the path is "images/" and we must set it to "messages/"
		/* "+ _en" is not needed here. It already exists. */
		finalName = path + name + ".txt" // construct the final TEXT name.
	}
	return finalName
}

func imgFileDataDecoder(imgFile *os.File) (image.Image, string, error) {
	/* In this function we're returning essentially the results of decoding the image into a golang image.Image object.	The results are:
	1. The image decoded,
	2. The registered function used to decode the image (the image type: gif|jpeg|png),
	3 Any errors during decoding */

	var (
		// imgType    string      // the file type (the extension) as returned by decoding.
		// imgDecoded image.Image // decoding image to image.Image interface.
		fileOffset int64 // the offset of the file
		err        error // error var.
	)

	fmt.Printf("Image File:\n")
	fileOffset, err = imgFile.Seek(0, 0)   // seek back the offset to 0, 0.
	checkError("File offset error: ", err) // check for errors.
	fmt.Printf("- File Offset is set to: %d Error: %v\n", fileOffset, err)

	// Create bufio.Reader object
	imgReader = bufio.NewReader(imgFile)
	// The type here is returned without the dot(.) e.g. "png"
	imgDecoded, imgType, err = image.Decode(imgReader)
	// Retrieves the size of the encoded message the image can store.
	sizeOfMessage = steganography.GetMessageSizeFromImage(imgDecoded)
	fmt.Printf("The image %q may hold a message of: %s bytes (~ %s)\n", imgFile.Name(), humanize.Comma(int64(sizeOfMessage)), humanize.Bytes(uint64(sizeOfMessage)))

	/* About file: */
	// Find the image type.
	fmt.Printf("The format name used during the format registration,\nand will be used during encoding of the new image, is: %v\n", imgType)
	switch imgType {
	case "gif":
		fmt.Println("GIF SELECTED") // image decoded with gif decoder.
	case "jpeg":
		fmt.Println("JPEG SELECTED") // image decoded with jpeg decoder.
	case "png":
		fmt.Println("PNG SELECTED") // image decoded with png decoder.
	default:
		fmt.Printf("The image type %v is not supported\n", imgType)
		usage()    // call usage
		os.Exit(1) // exit.
	}
	checkError("Error decoding image information: ", err) // check for errors.
	return imgDecoded, imgType, err
}

func checkError(errMes string, err error) {
	if err != nil {
		log.Fatalf(errMes, err)
	}
}
