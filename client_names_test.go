package skelc_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.yorun.ai/skelc"
)

func TestRuntimeClientLocalNameCollisions(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "service.skel")
	source := `domain demo.names
pub service NameService {
    method get {
        input {
            client: string
            ret: string
            err: string
            retI: string
            errI: string
        }
        output string
    }
    method find {
        output string?
    }
    method ping {
        input { err: string }
    }
}
`
	if err := os.WriteFile(input, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "generated")
	if _, err := skelc.CompileGolang(skelc.Input{SkelIn: input}, skelc.GolangOption{
		PubOnly:  true,
		AsModule: true,
		Module:   "example.com/names",
		Out:      output,
	}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = output
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, result)
		}
	}
}
