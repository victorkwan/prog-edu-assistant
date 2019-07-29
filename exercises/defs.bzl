# TODO(salikh): Implement the automatic tar rules too
def assignment_notebook(
	name,
	srcs,
	language = None,
	visibility = ["//visibility:private"]):
    """
    Defines a rule for student notebook and autograder
    generation from a master notebook.

    Arguments:
    name:
    srcs: the file name of the input notebook should end in '-master.ipynb'.
    """
    language_opt = ""
    language_suffix = ""
    if language:
      language_opt = " --language=" + language
      language_suffix = "-" + language
    native.genrule(
	name = name + "_student",
	srcs = srcs,
	outs = [srcs[0].replace('-master.ipynb','') + language_suffix + '-student.ipynb'],
	cmd = """$(location //go/cmd/assign) --input="$<" --output="$@" --command=student""" + language_opt,
	tools = [
	    "//go/cmd/assign",
	],
    )
    autograder_output = srcs[0].replace('-master.ipynb','') + language_suffix + '-autograder'
    native.genrule(
	name = name + "_autograder",
	srcs = srcs,
	outs = [autograder_output],
	cmd = """$(location //go/cmd/assign) --input="$<" --output="$@" --command=autograder""" + language_opt,
	tools = [
	    "//go/cmd/assign",
	],
    )
