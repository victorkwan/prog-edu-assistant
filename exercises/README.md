# Programming exercises

This directory contains the programming exercises and their autograder scripts
together.

## Installation of the client environment

TODO(salikh): Provide a simpler version of installation instructions for the
student environment.

Install virtualenv. The command may differ depending on the system.

    # On Debian or Ubuntu linux
    apt-get install python-virtualenv  # Install virtualenv.

    # On MacOS with Homebrew:
    brew install python3        # Make sure python3 is installed.
    pip3 install virtualenv     # Install virtualenv.

After that the setup procedure is common

    virtualenv -p python3 venv  # Create the virtual Python environment in ./venv/
    source ./venv/bin/activate  # Activate it.
    pip install jupyter         # Install Jupyter (inside of ./venv).

To start the Jupyter notebook run command

    jupyter notebook

## Structure of the programming assignment notebooks

Each programming assignment resides in a separate master Jupyter notebook. At
build time, the master notebook is taken as an input and the following outputs
are generated:

*   Student notebook
*   Autograder test directory
*   Automated tests for the notebook
    *   Testing master solution against student tests
    *   Testing master solution against autograder scripts
    *   Testing autograder scripts agains a variety of incomplete and incorrect
        solutions

A student notebook, and by extension, the source master notebook should contain
the following:

*   Explanation of a new concept, algorithm or library
*   Examples of use
*   Explanation of the tasks that the students should complete
*   An solution cell.
    *   In the student notebook the solution is replaced with a prompt of the
        form `... your solution here ...` or similar.
*   A few cells with tests for the student's solution, typically with built-in
    `assert` statements. These are used in two ways:
    *   To test the solution in the master notebook.
    *   To give students a few tests to check their solution.

All programming assignments are in one global shared namespace, where the
assignment notebook may have a few translations to different languages, denoted
by the suffix of the notebook, e.g. "en" for English and "ja" for Japanese. The
requirement for the assignment and exercise IDs is to be file name compatible.
The assignement name may include a course name to make the name globally unique.

Each student notebook should have a `assignment_id` entry in the notebook
metadata section that identifies the specific assignment and the course that the
assignment belongs to.

    "metadata": {
      "assignment_id": "HellowWorld",
      # ...
    },

This is useful for deciding which assignment the uploaded notebook is for and
for picking the correct autograder script to run. The metadata is provided in
the master notebook using triple-backtick sections with regexp-friendly markers
in YAML format (which means that the marker itself becomes a YAML comment and is
ignored). `# ASSIGNMENT METADATA` is copied into the notebook-level metadata
field of the student notebook, and `# EXERCISE METADATA` is copied into the cell
level metadata of the next code cell, which designates it as a _solution cell_.

    ```
    # ASSIGNMENT METADATA
    assignment_id: "HelloWorld"
    ```

    ```
    # EXERCISE METADATA
    exercise_id: "hello1"
    ```

The solution cell in the master notebook should contain the master solution,
marked with `BEGIN SOLUTION` and `END SOLUTION` markers:

    # BEGIN SOLUTION
		print("Hello, world")
    # END SOLUTION

The master solution will be replaced with `...` in the student notebook. If a
different replacement is desired, `BEGIN PROMPT` and `END PROMPT` markers may be used _before_ the SOLUTION block:

    """ # BEGIN PROMPT
    # put your program here
		pass
		""" # END PROMPT
    # BEGIN SOLUTION
		print("Hello, world")
    # END SOLUTION

The cells that contain student-oriented tests should be marked with `TEST`.
These typically using Python's `assert` builtin.

TODO(salikh): Remove the `# TEST` marker in student notebook.

TODO(salikh): Automatically extract `# TEST` cells as unit tests for the
master notebook.

The cells that are autograder scripts should be structured as standard Python unit tests using the `unittest` module. They need to have markers `BEGIN UNITTEST` and `END UNITTEST`. Only the lines between the markers are extracted into autograder scripts.
The preamble before `BEGIN UNITTEST` is useful to set up the environment in a manner
compatible with autograder environment, where 'import submission' is prepended.
In the notebook the recommended way is to use ad-hoc objects:

    from types import SimpleNamespace
    submission = SimpleNamespace(printHello=printHello)
	
The part of the cell after the `END UNITTEST` marker is also not written to
autograder scripts. It is useful to run the tests in the notebook inline, e.g.

    import sys
    import io
    suite = unittest.TestLoader().loadTestsFromTestCase(HelloOutputTest)
    errors = io.StringIO()
		# TODO(salikh): Move SummaryTestResult into a library and make it installable
		# via pip.
    result = unittest.TextTestRunner(verbosity=4,stream=errors, resultclass=SummaryTestResult).run(suite)
    # Optional.\n",
    #print(errors.getvalue())

    # TODO: Add some assertion on detected outcomes.
    print(result.results)

## Autograder tests

TODO(salikh): Define the syntax and the way to run autograder tests, i.e. the tests
that provide incomplete or incorrect input on purpose and check that the autograder
scripts (unit tests defined above in `UNITTEST` cells) produce expected combinations
of outcomes.

		# BEGIN AUTOTEST
		...
		# END AUTOTEST


## Structure of autograder scripts

NOTE: This is a proposed format that is subject to discussion and change.

The autograder scripts have two representations: the directory format and the
notebook format. The notebook format is the authoritative source and is
contained in the master notebook. The directory format is produced at build time
and is included into the autograder image, as well for automated testing of the
notebooks.

In the directory format, all autograder scripts take the form of python unit
tests (`*_test.py` files) runnable by the unittest runner. The student's
submission or the master solution will be written into a `submission.py` file
into the same directory (actually directory will be constructed using
overlayfs).

There may be a few special scripts, e.g. `extract.py` and `report.py` to extract
the student submission from submitted blob (typically extract the souce code of
one Jupyter notebook cell) and to convert a vector of test outcomes into a
human-readable report respectively.

## List of the exercises

*   `helloworld-en-master.ipynb` --- an example master assignment notebook to
    demonstrate the syntax.

## Request for contributions

Please add more exercises to this directory!
