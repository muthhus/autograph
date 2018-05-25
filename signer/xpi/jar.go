package xpi

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)


// makeJARManifestAndSignature writes hashes for all entries in a zip to a
// manifest file then hashes the manifest file to write a signature
// file and returns both
func makeJARManifestAndSignature(input []byte) (manifest, sigfile []byte, err error) {
	manifest, err = makeJARManifest(input)
	if err != nil {
		return
	}

	sigfile, err = makeJARSignature(manifest)
	if err != nil {
		return
	}

	return
}

// makeJARManifest calculates a sha1 and sha256 hash for each zip entry and writes them to a manifest file
func makeJARManifest(input []byte) (manifest []byte, err error) {
	inputReader := bytes.NewReader(input)
	r, err := zip.NewReader(inputReader, int64(len(input)))
	if err != nil {
		return
	}

	// generate the manifest file by calculating a sha1 and sha256 hash for each zip entry
	mw := bytes.NewBuffer(manifest)
	manifest = []byte(fmt.Sprintf("Manifest-Version: 1.0\n\n"))

	for _, f := range r.File {
		if isSignatureFile(f.Name) {
			// reserved signature files do not get included in the manifest
			continue
		}
		if f.FileInfo().IsDir() {
			// directories do not get included
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return manifest, err
		}
		data, err := ioutil.ReadAll(rc)
		if err != nil {
			return manifest, err
		}
		fmt.Fprintf(mw, "Name: %s\nDigest-Algorithms: SHA1 SHA256\n", f.Name)
		h1 := sha1.New()
		h1.Write(data)
		fmt.Fprintf(mw, "SHA1-Digest: %s\n", base64.StdEncoding.EncodeToString(h1.Sum(nil)))
		h2 := sha256.New()
		h2.Write(data)
		fmt.Fprintf(mw, "SHA256-Digest: %s\n\n", base64.StdEncoding.EncodeToString(h2.Sum(nil)))
	}
	manifestBody := mw.Bytes()
	manifest = append(manifest, manifestBody...)

	return
}

// makeJARSignature calculates a signature file by hashing the manifest with sha1 and sha256
func makeJARSignature(manifest []byte) (sigfile []byte, err error) {
	sw := bytes.NewBuffer(sigfile)
	fmt.Fprint(sw, "Signature-Version: 1.0\n")
	h1 := sha1.New()
	h1.Write(manifest)
	fmt.Fprintf(sw, "SHA1-Digest-Manifest: %s\n", base64.StdEncoding.EncodeToString(h1.Sum(nil)))
	h2 := sha256.New()
	h2.Write(manifest)
	fmt.Fprintf(sw, "SHA256-Digest-Manifest: %s\n\n", base64.StdEncoding.EncodeToString(h2.Sum(nil)))
	sigfile = sw.Bytes()

	return
}

// Metafile is a file to pack into a JAR at .Name with contents .Body
// .Name should begin with META-INF/ but this is not checked
type Metafile struct {
	Name string
	Body []byte
}

// repackJARWithMetafiles inserts metafiles in the input JAR file and returns a JAR ZIP archive
func repackJARWithMetafiles(input []byte, metafiles []Metafile) (output []byte, err error) {
	var (
		rc     io.ReadCloser
		fwhead *zip.FileHeader
		fw     io.Writer
		data   []byte
	)
	inputReader := bytes.NewReader(input)
	r, err := zip.NewReader(inputReader, int64(len(input)))
	if err != nil {
		return
	}
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	// Iterate through the files in the archive,
	for _, f := range r.File {
		// skip signature files, we have new ones we'll add at the end
		if isSignatureFile(f.Name) {
			continue
		}
		rc, err = f.Open()
		if err != nil {
			return
		}
		fwhead := &zip.FileHeader{
			Name:   f.Name,
			Method: zip.Deflate,
		}
		// insert the file into the archive
		fw, err = w.CreateHeader(fwhead)
		if err != nil {
			return
		}
		data, err = ioutil.ReadAll(rc)
		if err != nil {
			return
		}
		_, err = fw.Write(data)
		if err != nil {
			return
		}
		rc.Close()
	}
	// insert the signature files. Those will be compressed
	// so we don't have to worry about their alignment
	for _, meta := range metafiles {
		fwhead = &zip.FileHeader{
			Name:   meta.Name,
			Method: zip.Deflate,
		}
		fw, err = w.CreateHeader(fwhead)
		if err != nil {
			return
		}
		_, err = fw.Write(meta.Body)
		if err != nil {
			return
		}
	}
	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		return
	}

	output = buf.Bytes()
	return
}

// repackJAR inserts the manifest, signature file and pkcs7 signature in the input JAR file,
// and return a JAR ZIP archive
func repackJAR(input, manifest, sigfile, signature []byte) (output []byte, err error) {
	var metas = []Metafile{
		{"META-INF/manifest.mf", manifest},
		{"META-INF/mozilla.sf", sigfile},
		{"META-INF/mozilla.rsa", signature},
	}
	return repackJARWithMetafiles(input, metas)
}

// The JAR format defines a number of signature files stored under the META-INF directory
// META-INF/MANIFEST.MF
// META-INF/*.SF
// META-INF/*.DSA
// META-INF/*.RSA
// META-INF/SIG-*
// and their lowercase variants
// https://docs.oracle.com/javase/8/docs/technotes/guides/jar/jar.html#Signed_JAR_File
func isSignatureFile(name string) bool {
	if strings.HasPrefix(name, "META-INF/") {
		name = strings.TrimPrefix(name, "META-INF/")
		if name == "MANIFEST.MF" || name == "manifest.mf" ||
			strings.HasSuffix(name, ".SF") || strings.HasSuffix(name, ".sf") ||
			strings.HasSuffix(name, ".RSA") || strings.HasSuffix(name, ".rsa") ||
			strings.HasSuffix(name, ".DSA") || strings.HasSuffix(name, ".dsa") ||
			strings.HasPrefix(name, "SIG-") || strings.HasPrefix(name, "sig-") {
			return true
		}
	}
	return false
}

// mustReadFileFromZIP reads a given filename out of a ZIP and returns it or panics
func mustReadFileFromZIP(signedXPI []byte, filename string) (data []byte) {
	zipReader := bytes.NewReader(signedXPI)
	r, err := zip.NewReader(zipReader, int64(len(signedXPI)))
	if err != nil {
		panic(fmt.Sprintf("Error reading ZIP %s", err))
	}

	for _, f := range r.File {
		if f.Name == filename {
			rc, err := f.Open()
			defer rc.Close()
			if err != nil {
				panic(fmt.Sprintf("Error opening file %s in ZIP %s", filename, err))
			}
			data, err = ioutil.ReadAll(rc)
			if err != nil {
				panic(fmt.Sprintf("Error reading file %s in ZIP %s", filename, err))
			}
			return
		}
	}
	panic(fmt.Sprintf("failed to find %s in ZIP", filename))
}
