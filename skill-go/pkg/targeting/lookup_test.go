package targeting

import (
	"fmt"
	"math"
	"testing"
)

func TestImplicitTargetInfo_Accessors(t *testing.T) {
	tests := []struct {
		target  uint16
		objType ObjectTypes
		refType ReferenceTypes
		cat     SelectionCategory
		check   CheckTypes
		dirType DirectionTypes
	}{
		{1, ObjUnit, RefCaster, SelectDefault, CheckDefault, DirNone},   // TARGET_UNIT_CASTER
		{2, ObjUnit, RefCaster, SelectNearby, CheckEnemy, DirNone},      // TARGET_UNIT_NEARBY_ENEMY
		{6, ObjUnit, RefTarget, SelectDefault, CheckEnemy, DirNone},     // TARGET_UNIT_TARGET_ENEMY
		{15, ObjUnit, RefSrc, SelectArea, CheckEnemy, DirNone},          // TARGET_UNIT_SRC_AREA_ENEMY
		{24, ObjUnit, RefCaster, SelectCone, CheckEnemy, DirFront},      // TARGET_UNIT_CONE_ENEMY_24
		{47, ObjDest, RefCaster, SelectDefault, CheckDefault, DirFront}, // TARGET_DEST_CASTER_FRONT
		{54, ObjUnit, RefCaster, SelectCone, CheckEnemy, DirFront},      // TARGET_UNIT_CONE_180_DEG_ENEMY
		{76, ObjDest, RefCaster, SelectChannel, CheckDefault, DirNone},  // TARGET_DEST_CHANNEL_TARGET
		{77, ObjUnit, RefCaster, SelectChannel, CheckDefault, DirNone},  // TARGET_UNIT_CHANNEL_TARGET
		{133, ObjUnit, RefDest, SelectLine, CheckAlly, DirNone},         // TARGET_UNIT_LINE_CASTER_TO_DEST_ALLY
		{134, ObjUnit, RefDest, SelectLine, CheckEnemy, DirNone},        // TARGET_UNIT_LINE_CASTER_TO_DEST_ENEMY
	}

	for _, tt := range tests {
		iti := NewImplicitTargetInfo(tt.target)
		if got := iti.GetObjectType(); got != tt.objType {
			t.Errorf("target=%d GetObjectType() = %v, want %v", tt.target, got, tt.objType)
		}
		if got := iti.GetReferenceType(); got != tt.refType {
			t.Errorf("target=%d GetReferenceType() = %v, want %v", tt.target, got, tt.refType)
		}
		if got := iti.GetSelectionCategory(); got != tt.cat {
			t.Errorf("target=%d GetSelectionCategory() = %v, want %v", tt.target, got, tt.cat)
		}
		if got := iti.GetCheckType(); got != tt.check {
			t.Errorf("target=%d GetCheckType() = %v, want %v", tt.target, got, tt.check)
		}
		if got := iti.GetDirectionType(); got != tt.dirType {
			t.Errorf("target=%d GetDirectionType() = %v, want %v", tt.target, got, tt.dirType)
		}
	}
}

func TestImplicitTargetInfo_OutOfBounds(t *testing.T) {
	iti := NewImplicitTargetInfo(MaxImplicitTarget + 10)
	if got := iti.GetObjectType(); got != ObjNone {
		t.Errorf("out-of-bounds GetObjectType() = %v, want ObjNone", got)
	}
	if got := iti.GetReferenceType(); got != RefNone {
		t.Errorf("out-of-bounds GetReferenceType() = %v, want RefNone", got)
	}
	if got := iti.GetSelectionCategory(); got != SelectNYI {
		t.Errorf("out-of-bounds GetSelectionCategory() = %v, want SelectNYI", got)
	}
	if got := iti.GetCheckType(); got != CheckDefault {
		t.Errorf("out-of-bounds GetCheckType() = %v, want CheckDefault", got)
	}
	if got := iti.GetDirectionType(); got != DirNone {
		t.Errorf("out-of-bounds GetDirectionType() = %v, want DirNone", got)
	}
}

func TestImplicitTargetInfo_NYI(t *testing.T) {
	iti := NewImplicitTargetInfo(10) // index 10 not set → NYI
	if got := iti.GetSelectionCategory(); got != SelectNYI {
		t.Errorf("unset index GetSelectionCategory() = %v, want SelectNYI", got)
	}
	if got := iti.GetObjectType(); got != ObjNone {
		t.Errorf("unset index GetObjectType() = %v, want ObjNone", got)
	}
}

func TestImplicitTargetInfo_GetTarget(t *testing.T) {
	iti := NewImplicitTargetInfo(42)
	if got := iti.GetTarget(); got != 42 {
		t.Errorf("GetTarget() = %v, want 42", got)
	}
}

func TestImplicitTargetInfo_CalcDirectionAngle(t *testing.T) {
	tests := []struct {
		dir  DirectionTypes
		want float64
	}{
		{DirFront, 0.0},
		{DirBack, math.Pi},
		{DirRight, -math.Pi / 2},
		{DirLeft, math.Pi / 2},
		{DirFrontRight, -math.Pi / 4},
		{DirBackRight, -3 * math.Pi / 4},
		{DirBackLeft, 3 * math.Pi / 4},
		{DirFrontLeft, math.Pi / 4},
		{DirNone, 0.0},
	}

	for _, tt := range tests {
		// Find a target ID that maps to the desired direction
		// Use target 47 (DirFront) as base; CalcDirectionAngle reads from GetDirectionType
		// We'll test by creating an ImplicitTargetInfo with the right direction
		// Since we can't set direction directly, test with known entries
		t.Run(fmt.Sprintf("dir=%d", tt.dir), func(t *testing.T) {
			// Test the exported CalcDirectionAngle function directly
			got := CalcDirectionAngle(tt.dir)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("CalcDirectionAngle(%v) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestImplicitTargetInfo_CalcDirectionAngle_Random(t *testing.T) {
	// DirRandom should return [0, 2π)
	iti := NewImplicitTargetInfo(72) // TARGET_DEST_CASTER_RANDOM → DirRandom
	angle := iti.CalcDirectionAngle()
	if angle < 0 || angle >= 2*math.Pi {
		t.Errorf("DirRandom angle = %v, want in [0, 2π)", angle)
	}
}

func TestImplicitTargetInfo_IsArea(t *testing.T) {
	tests := []struct {
		target uint16
		want   bool
	}{
		{15, true},  // TARGET_UNIT_SRC_AREA_ENEMY → SelectArea
		{24, true},  // TARGET_UNIT_CONE_ENEMY_24 → SelectCone
		{1, false},  // TARGET_UNIT_CASTER → SelectDefault
		{2, false},  // TARGET_UNIT_NEARBY_ENEMY → SelectNearby
		{77, false}, // TARGET_UNIT_CHANNEL_TARGET → SelectChannel
	}

	for _, tt := range tests {
		iti := NewImplicitTargetInfo(tt.target)
		if got := iti.IsArea(); got != tt.want {
			t.Errorf("target=%d IsArea() = %v, want %v", tt.target, got, tt.want)
		}
	}
}
