# Student notebook repo

This repository is not a true source repository. It is automatically
generated from https://github.com/salikh/prog-edu-assistant.

## Local environment setup

### Conda

TODO: Add instructions for Conda.

### Binder

TODO: Add more detailed instructions for Binder.

You can start a notebook by visiting http://mybinder.org or directly
an URL like this:
https://mybinder.org/v2/gh/salikh/student-notebooks/master?filepath=functional-ja-student.ipynb

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
