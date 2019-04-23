// Package autograder provides the logic to parse the Jupyter notebook submissions,
// extract the assignment ID, match the assignment to the autograder scripts,
// set up the scratch directory and run the autograder tests under nsjail.
package autograder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/prog-edu-assistant/notebook"
	"github.com/golang/glog"
)

// Autograder encapsulates the setup of autograder scripts.
type Autograder struct {
	// Dir points to the root directory of autograder scripts.
	// Under Dir, the first level directory names are matched to assignment_id,
	// second level to exercise_id. In the second-level directories,
	// python unit test files should be present.
	Dir string
	NSJailPath string
}

// New creates a new autograder instance given the root directory.
func New(dir string) *Autograder {
	return &Autograder{
		Dir: dir,
	}
}

// Grade takes a byte blob, tries to parse it as JSON, then tries to extract
// the metadata and match it to the available corpus of autograder scripts.
// If found, it then proceeds to run all autograder scripts under nsjail,
// parse the output, and produce the report, also in JSON format.
func (ag *Autograder) Grade(notebookBytes []byte) ([]byte, error) {
	data := make(map[string]interface{})
	err := json.Unmarshal(notebookBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("could not parse request as JSON: %s", err)
	}
	v, ok := data["metadata"]
	if !ok {
		return nil, fmt.Errorf("request did not have .metadata")
	}
	metadata, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("metadata is not a map, but %s", reflect.TypeOf(v))
	}
	v, ok = metadata["submission_id"]
	if !ok {
		return nil, fmt.Errorf("request did not have submission_id")
	}
	submissionID, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("metadata.submission_id is not a string but %s",
			reflect.TypeOf(v))
	}
	v, ok = metadata["assignment_id"]
	if !ok {
		return nil, fmt.Errorf("metadata does not have assignment_id")
	}
	assignmentID, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("metadata.assignment_id is not a string but %s",
			reflect.TypeOf(v))
	}
	dir := filepath.Join(ag.Dir, assignmentID)
	glog.V(3).Infof("dir = %q", dir)
	fs, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("assignment with id %q does not exit", assignmentID)
	}
	if !fs.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dir)
	}
	n, err := notebook.Parse(notebookBytes)
	if err != nil {
		return nil, err
	}
	allOutcomes := make(map[string]bool)
	for _, cell := range n.Cells {
		if cell.Metadata == nil {
			continue
		}
		v, ok := cell.Metadata["exercise_id"]
		if !ok {
			continue
		}
		exerciseID, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("exercise_id is not a string but %s",
				reflect.TypeOf(v))
		}
		exerciseDir := filepath.Join(dir, exerciseID)
		fs, err = os.Stat(exerciseDir)
		if err != nil {
			return nil, fmt.Errorf("exercise with id %s/%s does not exit",
				assignmentID, exerciseID)
		}
		if !fs.IsDir() {
			return nil, fmt.Errorf("%q is not a directory", exerciseDir)
		}
		// TODO(salikh): Implement proper scratch management with overlayfs.
		filename := filepath.Join(exerciseDir, "submission.py")
		err := ioutil.WriteFile(filename, []byte(cell.Source), 0775)
		if err != nil {
			return nil, fmt.Errorf("error writing to %q: %s", filename, err)
		}
		outcomes, err := ag.RunUnitTests(exerciseDir)
		if err != nil {
			return nil, fmt.Errorf("error running unit tests in %q: %s", exerciseDir, err)
		}
		for k, v := range outcomes {
			_, ok := allOutcomes[k]
			if ok {
				return nil, fmt.Errorf("duplicated unit test %q", k)
			}
			allOutcomes[k] = v
		}
	}
	result := make(map[string]interface{})
	result["assignment_id"] = assignmentID
	result["submission_id"] = submissionID
	result["outcomes"] = allOutcomes
	b, err := json.Marshal(allOutcomes)
	if err != nil {
		return nil, fmt.Errorf("error serializing report json: %s", err)
	}
	return b, nil
}

// nsjail -Mo --time_limit 2 --max_cpus 1 --rlimit_as 700 -E LANG=en_US.UTF-8 --disable_proc --chroot / --cwd $PWD --user nobody --group nogroup --iface_no_lo -- /usr/bin/python3 -m unittest discover -v -p '*_test.py'

var outcomeRegex = regexp.MustCompile(`(test[a-zA-Z0-9_]*) \(([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_]*)\) \.\.\. (ok|FAIL|ERROR)`)

func (ag *Autograder) RunUnitTests(dir string) (map[string]bool, error) {
	err := os.Chdir(dir)
	if err != nil {
		return nil, fmt.Errorf("error on chdir %q: %s", dir, err)
	}
	fss, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error on listing %q: %s", dir, err)
	}
	outcomes := make(map[string]bool)
	for _, fs := range fss {
		filename := fs.Name()
		if !strings.HasSuffix(filename, "_test.py") {
			continue
		}
		cmd := exec.Command(ag.NSJailPath, "-Mo",
			"--time_limit", "3",
			"--max_cpus", "1",
			"--rlimit_as", "700",
			"-E", "LANG=en_US.UTF-8",
			"--disable_proc",
			"--chroot", "/",
			"--cwd", dir,
			"--user", "nobody",
			"--group", "nogroup",
			"--iface_no_lo",
			"--",
			"/usr/bin/python3", "-m", "unittest",
			"-v", fs.Name())
		out, err := cmd.CombinedOutput()
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return nil, fmt.Errorf("error running unit test command %q: %s", cmd, err)
			}
			// Overall status was non-ok.
			outcomes[filename] = false
		} else {
			// The file run okay.
			outcomes[filename] = true
		}
		// TODO(salikh): Implement a more robust way of reporting individual
		// test statuses from inside the test runner.
		mm := outcomeRegex.FindAllSubmatch(out, -1)
		if len(mm) == 0 {
			// Cannot find any individual test case outcomes.
			outcomes[filename] = false
			continue
		}
		for _, m := range mm {
			method := string(m[1])
			className := string(m[3])
			status := string(m[4])
			key := className + "." + method
			if status == "ok" {
				outcomes[key] = true
			} else {
				outcomes[key] = false
			}
		}
	}
	return outcomes, nil
}
