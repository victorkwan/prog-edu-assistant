# Student notebook repo

This repository is not a true source repository. It is automatically
generated from https://github.com/salikh/prog-edu-assistant,
which itself is a fork of https://github.com/google/prog-edu-assistant.

## Binder

You can open notebooks from this repository by clicking the following links:
* Functional programming/OOP: https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=functional-ja-en-ja-student.ipynb
* Dataframes 1: https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=dataframe-pre1-en-ja-student.ipynb
* Dataframes 2: https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=dataframe-pre2-en-ja-student.ipynb
* Dataframes 3: https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=dataframe-pre3-en-ja-student.ipynb

## Local environment setup

### Conda

TODO: Add instructions for Conda.

### Virtualenv

If you use Debian-based Linux, use the following commands for local setup:

    apt-get install python-virtualenv
    virtualenv -p python3 ../venv
    source ../venv/bin/activate
    pip install -r requirements.txt

    jupyter nbextension install nbextensions/upload_it --symlink
    jupyter nbextension enable upload_it/main
    jupyter nbextensions_configurator enable --user

    jupyter notebook

## License

Apache-2.0; see [LICENSE](LICENSE) for details.

## Disclaimer

This project is not an official Google project. It is not
supported by Google and Google specifically disclaims all
warranties as to its quality, merchantability, or fitness for
a particular purpose.
