package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wgdl666/wgModelHub/internal/channel"
)

// 本地改写 providers 键和 model_routes 引用。只读写给出的文件，不连接配置中心，也不重启进程。
// 输出是一份带凭据的完整配置，所以只新建 0600 文件，拒绝覆盖已有路径，也拒绝写回输入路径。
func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "usage: rename-providers <input.yaml> <output.yaml>")
		return 2
	}
	input, output := args[0], args[1]
	same, err := samePath(input, output)
	if err != nil || same {
		fmt.Fprintln(stderr, "output path must be a new file")
		return 1
	}
	content, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintln(stderr, "read input failed")
		return 1
	}
	rewritten, renames, err := channel.RewriteYAML(content)
	if err != nil {
		fmt.Fprintln(stderr, "rewrite failed")
		return 1
	}
	if err := createExclusive(output, rewritten); err != nil {
		fmt.Fprintln(stderr, "write output failed")
		return 1
	}
	fmt.Fprintf(stderr, "renamed %d instances\n", len(renames))
	return 0
}

func samePath(left, right string) (bool, error) {
	leftPath, err := filepath.Abs(left)
	if err != nil {
		return false, err
	}
	rightPath, err := filepath.Abs(right)
	if err != nil {
		return false, err
	}
	return leftPath == rightPath, nil
}

func createExclusive(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return errors.Join(writeErr, closeErr)
	}
	return nil
}
