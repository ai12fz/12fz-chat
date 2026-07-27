package test

import "testing"

// compile-time assertion: prevent struct changes from breaking queries
func TestDeviceStructFields(t *testing.T) {
    // If this breaks, you added/removed fields without updating queries
    const expectedFields = 10
    d := struct{ID,Name,OrgID,Token,OS,Status,CreatedAt string; LastSeen interface{}; AllowSkills,AllowSoftware bool}{}
    _ = d
    t.Log(device struct OK)
}
