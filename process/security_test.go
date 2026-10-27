package process

import "testing"

func TestFindCmdsDoesNotSplitExpandedValues(t *testing.T) {
	t.Setenv("WE_TEST_INJECT", "x | rm -rf /tmp/nope")

	cmds := findCmds("echo $WE_TEST_INJECT")
	if len(cmds) != 1 {
		t.Fatalf("expanded value created %d pipeline stages, want 1", len(cmds))
	}
	want := []string{"echo", "x | rm -rf /tmp/nope"}
	if !compareArgs(cmds[0].Args, want) {
		t.Errorf("args = %q, want %q", cmds[0].Args, want)
	}
}

func TestFindCmdsPipeFromVariableIsLiteral(t *testing.T) {
	t.Setenv("WE_TEST_PIPE", "|")

	cmds := findCmds("echo $WE_TEST_PIPE cat")
	if len(cmds) != 1 {
		t.Fatalf("variable containing a pipe created %d stages, want 1", len(cmds))
	}
	want := []string{"echo", "|", "cat"}
	if !compareArgs(cmds[0].Args, want) {
		t.Errorf("args = %q, want %q", cmds[0].Args, want)
	}
}

func TestSplitCommandExpandsOncePerToken(t *testing.T) {
	t.Setenv("WE_TEST_A", "$WE_TEST_B")
	t.Setenv("WE_TEST_B", "nested")

	parts := SplitCommand("cmd $WE_TEST_A")
	want := []string{"cmd", "$WE_TEST_B"}
	if !compareArgs(parts, want) {
		t.Errorf("parts = %q, want %q", parts, want)
	}
}
