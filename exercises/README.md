# Programming exercises

This directory contains the programming exercises and their autograder scripts
together.

## Installation of the client environment

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
the master notebook using triple-backtick sections with regexp-friendly markers:

    ```
    # ASSIGNMENT METADATA
    assignment_id = "HelloWorld"
    ```

    ```
    # EXERCISE METADATA
    exercise_id = "hello1"
    ```

## Structure of autograder scripts

NOTE: This is a proposed format that is subject to discussion and change.

The autograder scripts have two representations: the directory format and the
notebook format. The notebook format is the authoritative source and is
contained in the master notebook. The directory format is produced at build time
and

The python files in the this directory with the basename matching the assignment
notebooks are autograder scripts.

Each autograder script is a python library that should expose the following
function:

    def Autograde(content, metadata)
      # Input:
      # * content - the uploaded content as a text string. Typically
      #             this is a JSON-encoded Jupyter notebook, but it may
      #             be a standalone python file or something else entirely
      #             depending on the specific assignment.
      # * metadata - optional JSON-encoded string with additional metadata
      #              that may have been provided with the upload, or None.
      #              E.g. this may be used to pass the file name of the
      #              upload, or to specify the assignment name explicitly.
      #
      # Returns: A JSON-encoded object with the result of analysis. It should
      # include:
      # * source - The source code that has been extracted as a students'
      #            solution. The intention is that this code may be presented
      #            to the student in the report.
      # * findings - A list of findings that specify a range (starting
      #              line/column and ending line/column) of the source
      #              code and the message. The intention is that these may
      #              be rendered as wavy underline or colored messages with
      #              details shown on mouse hover.
      # * messages - A list of general human-readable message to be presented
      #              to the student.

## List of the exercises

*   `helloworld-en.ipynb` --- a minimal exercise template

## Request for contributions

Please add more exercises to this directory!
