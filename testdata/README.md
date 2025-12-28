# README

This directory contains 4 files:
- app_dot_go_v1.txt
- app_dot_go_v2.txt
- expected_output.txt
- decolored_expected_output.txt

`app_dot_go_v1.txt` was created with the following command

`git show ad46a5c0d9f4b92d24d3a2c07570f65dd21483eb > app_dot_go_v1.txt`

`app_dot_go_v2.txt` was created with the following command

`git show 97cfec101131551437441f4bce21d04566161c33 > app_dot_go_v2.txt`

`expected_output.txt` was created by running the following bash code

```bash
dwdiff -A best -L -c -d "\x0A%,;/:._{}[]()'\!|-=~><\"\\\\" >
expected_output.txt
```

note that `expected_output.txt` contains ANSI escape codes. You should
be able to run `cat expected_output.txt` and see a colorized output
with line numbers in the left column.
