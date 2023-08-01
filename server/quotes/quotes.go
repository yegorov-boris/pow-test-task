package quotes

import (
	"bufio"
	"github.com/pkg/errors"
	"math/rand"
	"os"
)

type Generator struct {
	qs []string
}

func NewGenerator() (*Generator, error) {
	readFile, err := os.OpenFile("quotes.txt", os.O_RDONLY, 0744)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open the quotes file")
	}

	defer readFile.Close()

	g := &Generator{}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if len(line) != 0 {
			g.qs = append(g.qs, line)
		}
	}

	return g, nil
}

func (g *Generator) Get() string {
	return g.qs[rand.Intn(len(g.qs))]
}
