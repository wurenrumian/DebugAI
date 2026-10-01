package service

import (
	"testing"

	"backend-go/models"
)

func member(role string, creator bool) models.ClassMember {
	return models.ClassMember{MemberRole: role, IsCreator: creator}
}

func TestCanChangeMemberRole(t *testing.T) {
	const (
		teacher = models.MemberRoleTeacher
		ta      = models.MemberRoleTA
		student = models.MemberRoleStudent
	)
	cases := []struct {
		name          string
		operatorRole  string
		opIsCreator   bool
		opIsSysAdmin  bool
		target        models.ClassMember
		newRole       string
		expectAllowed bool
	}{
		{"nobody can change creator", teacher, true, true, member(teacher, true), student, false},
		{"TA cannot demote teacher", ta, false, false, member(teacher, false), student, false},
		{"TA cannot demote another TA", ta, false, false, member(ta, false), student, false},
		{"TA cannot promote student", ta, false, false, member(student, false), ta, false},
		{"non-creator teacher cannot demote teacher", teacher, false, false, member(teacher, false), student, false},
		{"non-creator sysadmin teacher cannot demote teacher", teacher, false, true, member(teacher, false), student, false},
		{"non-creator teacher can demote TA", teacher, false, false, member(ta, false), student, true},
		{"non-creator teacher cannot promote student", teacher, false, false, member(student, false), ta, false},
		{"sysadmin teacher can promote student", teacher, false, true, member(student, false), ta, true},
		{"creator can demote teacher", teacher, true, false, member(teacher, false), student, true},
		{"creator can promote student to teacher", teacher, true, false, member(student, false), teacher, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, reason := CanChangeMemberRole(tc.operatorRole, tc.opIsCreator, tc.opIsSysAdmin, tc.target, tc.newRole)
			if ok != tc.expectAllowed {
				t.Fatalf("got allowed=%v (reason=%q), want %v", ok, reason, tc.expectAllowed)
			}
			if !ok && reason == "" {
				t.Fatal("denial must include a reason")
			}
		})
	}
}

func TestCanRemoveMember(t *testing.T) {
	const (
		teacher = models.MemberRoleTeacher
		ta      = models.MemberRoleTA
		student = models.MemberRoleStudent
	)
	cases := []struct {
		name          string
		operatorRole  string
		opIsCreator   bool
		target        models.ClassMember
		expectAllowed bool
	}{
		{"creator cannot be removed", teacher, true, member(student, true), false},
		{"TA cannot remove teacher", ta, false, member(teacher, false), false},
		{"TA cannot remove TA", ta, false, member(ta, false), false},
		{"TA can remove student", ta, false, member(student, false), true},
		{"non-creator teacher cannot remove teacher", teacher, false, member(teacher, false), false},
		{"teacher can remove TA", teacher, false, member(ta, false), true},
		{"creator can remove teacher", teacher, true, member(teacher, false), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, _ := CanRemoveMember(tc.operatorRole, tc.opIsCreator, tc.target)
			if ok != tc.expectAllowed {
				t.Fatalf("got allowed=%v, want %v", ok, tc.expectAllowed)
			}
		})
	}
}
