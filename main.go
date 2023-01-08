package main

import (
	"bytes"
	_ "embed"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const dirPerm fs.FileMode = 0755
const fileParam fs.FileMode = 0666

//go:embed tmpl/cmpt.svelte.tmpl
var svelteTmpl string

//go:embed tmpl/index.html.tmpl
var htmlTmpl string

//go:embed tmpl/package.json.tmpl
var packageTmpl string

//go:embed tmpl/rollup.config.js.tmpl
var rollupTmpl string

//go:embed tmpl/gitignore.tmpl
var gitignoreTmpl string

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("missing cmpt dir")
	}

	cmptDir := os.Args[1]
	log.Printf("init cmpt %q", cmptDir)
	if e := initCmpt(cmptDir); e != nil {
		log.Fatalf("not able to init cmpt: %v", e)
	}
}

func createTmplFile(p string, tmpl string, cmptName string) error {
	t := template.Must(template.New(p).Parse(tmpl))
	b := &bytes.Buffer{}
	if e := t.Execute(b, struct{ CmptName string }{cmptName}); e != nil {
		return e
	}
	if e := os.WriteFile(p, b.Bytes(), 0660); e != nil {
		return e
	}
	return nil
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func runCmd(cmd *exec.Cmd) *bytes.Buffer {
	name := cmd.Args[0]
	outbuf := &bytes.Buffer{}
	cmd.Stdout = outbuf
	errbuf := &bytes.Buffer{}
	cmd.Stderr = errbuf

	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start() error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			log.Fatalf("cmd %s error: %v\n%s", name, e, errbuf.String())
		} else {
			log.Fatalf("cmd %s error: %v", name, err)
		}
	}
	return outbuf
}

func initCmpt(cmptDir string) (err error) {
	name := filepath.Base(cmptDir)
	log.Printf("+ init %q", name)
	if dirExists(cmptDir) {
		log.Fatalf("%q does already exist", cmptDir)
	}
	srcDir := filepath.Join(cmptDir, "src")
	err = os.MkdirAll(srcDir, 0750)
	if err != nil {
		return err
	}
	err = createTmplFile(filepath.Join(srcDir, name+".svelte"), svelteTmpl, name)
	if err != nil {
		return err
	}

	if e := os.WriteFile(filepath.Join(srcDir, "global.d.ts"), []byte(`/// <reference types="svelte" />`), fileParam); e != nil {
		return e
	}

	distDir := filepath.Join(cmptDir, "dist")
	err = os.MkdirAll(distDir, dirPerm)
	if err != nil {
		return err
	}
	err = createTmplFile(filepath.Join(distDir, "index.html"), htmlTmpl, name)
	if err != nil {
		return err
	}

	err = createTmplFile(filepath.Join(cmptDir, "package.json"), packageTmpl, name)
	if err != nil {
		return err
	}

	err = createTmplFile(filepath.Join(cmptDir, "rollup.config.js"), rollupTmpl, name)
	if err != nil {
		return err
	}

	err = createTmplFile(filepath.Join(cmptDir, ".gitignore"), gitignoreTmpl, name)
	if err != nil {
		return err
	}

	log.Printf("+ npm install for %q", name)
	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = cmptDir
	runCmd(npmInstall)

	return nil
}
