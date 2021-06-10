package membership

import (
	pb "github.com/authzed/spicedb/pkg/REDACTEDapi/api"
	"github.com/authzed/spicedb/pkg/tuple"
)

// MembershipSet represents the set of membership for one or more ONRs, based on expansion
// trees.
type MembershipSet struct {
	// objectsAndRelations is a map from an ONR (as a string) to the subjects found for that ONR.
	objectsAndRelations map[string]FoundSubjects
}

// SubjectsByONR returns a map from ONR (as a string) to the FoundSubjects for that ONR.
func (ms *MembershipSet) SubjectsByONR() map[string]FoundSubjects {
	return ms.objectsAndRelations
}

// FoundSubjects contains the subjects found for a specific ONR.
type FoundSubjects struct {
	// subjects is a map from the Subject ONR (as a string) to the FoundSubject information.
	subjects map[string]FoundSubject
}

// ListFound returns a slice of all the FoundSubject's.
func (fs FoundSubjects) ListFound() []FoundSubject {
	found := []FoundSubject{}
	for _, sub := range fs.subjects {
		found = append(found, sub)
	}
	return found
}

// LookupSubject returns the FoundSubject for a matching subject, if any.
func (fs FoundSubjects) LookupSubject(subject *pb.ObjectAndRelation) (FoundSubject, bool) {
	onrString := tuple.StringONR(subject)
	found, ok := fs.subjects[onrString]
	return found, ok
}

// FoundSubject contains a single found subject and all the relationships in which that subject
// is a member which were found via the ONRs expansion.
type FoundSubject struct {
	// subject is the subject found.
	subject *pb.ObjectAndRelation

	// relations are the relations under which the subject lives that informed the locating
	// of this subject for the root ONR.
	relationships *tuple.ONRSet
}

// Subject returns the Subject of the FoundSubject.
func (fs FoundSubject) Subject() *pb.ObjectAndRelation {
	return fs.subject
}

// Relationships returns all the relationships in which the subject was found as per the expand.
func (fs FoundSubject) Relationships() []*pb.ObjectAndRelation {
	return fs.relationships.AsSlice()
}

// NewMembershipSet constructs a new membership set.
//
// NOTE: This is designed solely for the developer API and should *not* be used in any performance
// sensitive code.
func NewMembershipSet() *MembershipSet {
	return &MembershipSet{
		objectsAndRelations: map[string]FoundSubjects{},
	}
}

// AddExpansion adds the expansion of an ONR to the membership set. Returns false if the ONR was already added.
//
// NOTE: The expansion tree *should* be the fully recursive expansion.
func (ms *MembershipSet) AddExpansion(onr *pb.ObjectAndRelation, expansion *pb.RelationTupleTreeNode) (FoundSubjects, bool) {
	onrString := tuple.StringONR(onr)
	existing, ok := ms.objectsAndRelations[onrString]
	if ok {
		return existing, false
	}

	foundSubjectsMap := map[string]FoundSubject{}
	ms.populateFoundSubjects(foundSubjectsMap, onr, expansion)

	fs := FoundSubjects{
		subjects: foundSubjectsMap,
	}
	ms.objectsAndRelations[onrString] = fs
	return fs, true
}

func (ms *MembershipSet) populateFoundSubjects(foundSubjectsMap map[string]FoundSubject, rootONR *pb.ObjectAndRelation, treeNode *pb.RelationTupleTreeNode) {
	relationship := rootONR
	if treeNode.Expanded != nil {
		relationship = treeNode.Expanded
	}

	switch typed := treeNode.NodeType.(type) {
	case *pb.RelationTupleTreeNode_IntermediateNode:
		switch typed.IntermediateNode.Operation {
		case pb.SetOperationUserset_UNION:
			fallthrough

		case pb.SetOperationUserset_INTERSECTION:
			fallthrough

		case pb.SetOperationUserset_EXCLUSION:
			for _, child := range typed.IntermediateNode.ChildNodes {
				ms.populateFoundSubjects(foundSubjectsMap, rootONR, child)
			}

		default:
			panic("unknown expand operation")
		}

	case *pb.RelationTupleTreeNode_LeafNode:
		for _, user := range typed.LeafNode.Users {
			subjectONRString := tuple.StringONR(user.GetUserset())
			_, ok := foundSubjectsMap[subjectONRString]
			if !ok {
				foundSubjectsMap[subjectONRString] = FoundSubject{
					subject:       user.GetUserset(),
					relationships: tuple.NewONRSet(),
				}
			}

			foundSubjectsMap[subjectONRString].relationships.Add(relationship)
		}
	default:
		panic("unknown TreeNode type")
	}
}
