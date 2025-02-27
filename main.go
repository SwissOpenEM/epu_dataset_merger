package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const maxWorkers = 8

func main() {

	oFlag := flag.String("o", "", "provide target path where output datasets (tif/eer/mrc) are written")
	iFlag := flag.String("i", "", "provide target path where EPU stores the metadata in mirrored folders (xmls, jpgs, mrcs)")
	aFlag := flag.String("a", "", "provide path to EPU Atlas folder if you want to also store the overview Atlas with your dataset.")

	flag.Parse()
	var xRoot string
	var yRoot string
	xtest, err := filepath.Abs(*oFlag)
	if err != nil {
		log.Fatalf("-o Flag has been set incorrectly, %s", err)
	} else {
		xRoot = xtest
		log.Println("Datasets:", xRoot)
	}
	ytest, err := filepath.Abs(*iFlag)
	if err != nil {
		log.Fatalf("-i Flag has been set incorrectly, %s", err)
	} else {
		yRoot = ytest
		log.Println("EPU folder:", yRoot)
	}

	if err := syncXMLFromYtoX(xRoot, yRoot, maxWorkers, *aFlag); err != nil {
		log.Fatalf("Error syncing XML files: %v", err)
	}
}

func syncXMLFromYtoX(xDir, yDir string, workers int, atlas string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tasks := make(chan copyTask, 1000) // buffered channel to reduce blocking

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			copyWorker(ctx, tasks)
		}()
	}
	contents, err := os.ReadDir(xDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldnt read --o", err)
	}
	for _, d := range contents {
		xPath, err := filepath.Abs(filepath.Join(xDir, d.Name()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Filepath couldnt be determined correctly , %s", err)
		}
		// Only process directories
		if !d.IsDir() {
			continue
		}
		relPath, err := filepath.Rel(xDir, xPath)
		if err != nil {
			return err
		}
		yPath := filepath.Join(yDir, relPath)
		info, err := os.Stat(yPath)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if !info.IsDir() {
			continue
		} //till here we now found a folder that exists in x and y  (dataset folder)
		// if aFLag set correctly, grab the Atlas from another TFS directory.
		fmt.Println("Currently working on project:", d.Name())
		if atlas != "" {
			err := getAtlas(yPath, atlas, xPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't obtain the Atlas because: %s \n", err)
			}
		}
		// Walk that subfolder in Y for .xml files.
		walkErr := filepath.WalkDir(yPath, func(path string, yd os.DirEntry, yerr error) error {
			if yerr != nil {
				if os.IsPermission(yerr) {
					return filepath.SkipDir
				}
				return yerr
			}
			if yd.IsDir() && yd.Name() == "FoilHoles" { // SPA/EPU skip
				return filepath.SkipDir
			} else if yd.IsDir() && (yd.Name() == "Thumbnails") { // Tomo skip
				return filepath.SkipDir
			} else if yd.IsDir() && (yd.Name() == "SearchMaps") { // this is where we need the whole folder
				target := filepath.Join(yPath, yd.Name())
				filepath.WalkDir(target, func(path string, yd os.DirEntry, yerr error) error {
					if yd.IsDir() { // start crawling, generate folders as we go
						var new_leg string
						if yd.Name() != "SearchMaps" {
							new_leg = filepath.Join(target, yd.Name())
						} else {
							new_leg = filepath.Join(yPath, yd.Name())
						}
						cross, _ := filepath.Rel(yPath, new_leg)
						newdir1 := filepath.Join(xPath, cross)
						_, direrr1 := os.Stat(newdir1)
						if os.IsNotExist(direrr1) {
							os.Mkdir(newdir1, 0755)
							fmt.Println("created", newdir1)
						}
						return nil
					} else {
						yRel, err := filepath.Rel(yPath, path)
						if err != nil {
							return err
						}
						xDestPath := filepath.Join(xPath, yRel)
						_, infoerr := os.Stat(xDestPath)
						if infoerr == nil {
							return nil
						}
						select {
						case tasks <- copyTask{src: path, dst: xDestPath}:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					return nil
				})

				return filepath.SkipDir
			} else if yd.IsDir() && (yd.Name() == "Batch") {
				newdir := filepath.Join(xPath, "Batch")
				_, direrr := os.Stat(newdir)
				if os.IsNotExist(direrr) {
					os.Mkdir(newdir, 0755)
				} // make sure the Folder is created in X to be able to copy to it.
				return nil
			} else if yd.IsDir() { // want Data in EPU/SPA
				return nil
			}
			// Only process .xml files
			if strings.HasSuffix(strings.ToLower(yd.Name()), ".mdoc") {
				yRel, err := filepath.Rel(yPath, path)
				if err != nil {
					return err
				}
				// Grab a special case in Tomo5 data where a y Flip occurs depending on the output file format, and leave a small text file to report on it.
				xDestPath := filepath.Join(xPath, yRel)
				message := filepath.Join(filepath.Dir(xDestPath), "HowToFlipMyTomoData.txt")
				_, errmessage := os.Stat(message)
				if !(errmessage == nil) {
					pattern := `^Position_.*$`
					re := regexp.MustCompile(pattern)
					test, errtest := os.ReadDir(filepath.Dir(message))
					if errtest != nil {
						fmt.Fprintln(os.Stderr, "cant read directory", errtest)
					}
					for _, name := range test {
						file := name.Name()
						if re.Match([]byte(file)) {
							switch filepath.Ext(file) {
							case ".tiff":
								data := []byte("This data was originally written as tiff by Tomo5!\n")
								errw := os.WriteFile(message, data, 0644)
								if errw != nil {
									fmt.Fprintln(os.Stderr, "Printing flip instructions went wrong")
								}

							case ".eer":
								data := []byte("This data is orginially written as eer by Tomo5!\n")
								errw := os.WriteFile(message, data, 0644)
								if errw != nil {
									fmt.Fprintln(os.Stderr, "Printing flip instructions went wrong")
								}
							}
						}

					}
				}
				return nil

			} else if !strings.HasSuffix(strings.ToLower(yd.Name()), ".xml") {
				return nil
			}
			if strings.Contains(yd.Name(), "GridSquare") {
				return nil
			}
			yRel, err := filepath.Rel(yPath, path)
			if err != nil {
				return err
			}
			xDestPath := filepath.Join(xPath, yRel)
			_, infoerr := os.Stat(xDestPath)
			if infoerr == nil {
				return nil
			}
			select {
			case tasks <- copyTask{src: path, dst: xDestPath}:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})
		if walkErr != nil && walkErr != filepath.SkipDir {
			return walkErr
		}

	}
	close(tasks)
	wg.Wait()
	return nil
}

// queue copy function
func copyWorker(ctx context.Context, tasks <-chan copyTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			if err := copyFile(task.src, task.dst); err != nil {
				log.Printf("copy error: %v", err)
			}
		}
	}
}

type copyTask struct {
	src string
	dst string
}

// actual copy function
func copyFile(src, dst string) error {

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open src %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create dst %s: %w", dst, err)
	}

	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy from %s to %s: %w", src, dst, err)
	}

	return err
}

// Func to grab the atlas if aFlag is set - fully optional, if set will copy the overall Atlas.mrc in the top level dataset directory
func getAtlas(yPath string, atlas string, xPath string) error {
	contents, err := os.ReadDir(yPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldnt read path during Atlas search", err)
	}
	for _, d := range contents {
		if d.Name() == "EpuSession.dm" || d.Name() == "Session.dm" {
			sessionfile := yPath + string(filepath.Separator) + d.Name()
			f, err := os.Open(sessionfile)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			elementName := "AtlasId"
			value, err := findAtlasIDValue(f, elementName)
			if err != nil {
				fmt.Println("Error:", err)
				return err
			}
			// Change the Windows path to a linux one (if required) and change drive D:\ start to mountpoint *aFlag
			atlasdm := strings.ReplaceAll(value, `\`, string(filepath.Separator))
			if len(atlasdm) > 1 && atlasdm[1] == ':' {
				atlasdm = atlasdm[2:]
			}
			full := atlas + atlasdm
			targetpath := filepath.Dir(full)
			atlas_re := `^Atlas_.*\.mrc$`
			find := regexp.MustCompile(atlas_re)
			test, errtest := os.ReadDir(targetpath)
			if errtest != nil {
				return errtest
			}
			for _, name := range test {
				file := name.Name()
				if find.Match([]byte(file)) {
					doublecopyprotect := xPath + string(filepath.Separator) + file
					atlasfile := targetpath + string(filepath.Separator) + file
					_, checkerr := os.Stat(doublecopyprotect)
					switch checkerr {
					case nil:
						continue
					default:
						err := copyFile(atlasfile, doublecopyprotect)
						if err != nil {
							return err
						}
					}

				}
			}
		}
	}
	return nil
}

// extract the Atlas Folder Path from the Session xml (thermo: .dm) file
func findAtlasIDValue(r io.Reader, elementName string) (string, error) {
	decoder := xml.NewDecoder(r)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return "", fmt.Errorf("element %q not found in Session", elementName)
			}
			return "", err
		}
		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == elementName {
				var val string
				err := decoder.DecodeElement(&val, &se)
				if err != nil {
					return "", err
				}
				return val, nil
			}
		}
	}
}
