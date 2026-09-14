package localintel

import "testing"

func TestMinSufficientRouter(t *testing.T) {
	r := MinSufficientRouter()
	pkgs := []ModelPackage{
		{Name: "z-gen", Runtime: "llama.cpp"},
		{Name: "a-emb", Runtime: "onnx"},
	}
	got, err := r.Route(KnowledgeTask{ID: "t1", Kind: TaskEmbedding}, pkgs)
	if err != nil || got.Name != "a-emb" {
		t.Fatalf("embedding must route to onnx: %+v %v", got, err)
	}
	got, err = r.Route(KnowledgeTask{ID: "t2", Kind: TaskGeneration}, pkgs)
	if err != nil || got.Name != "z-gen" {
		t.Fatalf("generation must route to llama.cpp: %+v %v", got, err)
	}
	if _, err := r.Route(KnowledgeTask{ID: "t3", Kind: "bogus"}, pkgs); err == nil {
		t.Fatal("unknown kind must error")
	}
	if _, err := r.Route(KnowledgeTask{ID: "t4", Kind: TaskEmbedding}, nil); err == nil {
		t.Fatal("empty pool must error, never hallucinate a model")
	}
}

func TestProfilesArePolicies(t *testing.T) {
	for _, p := range []Profile{ProfileNoLLM, ProfileUltraLight, ProfileLight, ProfileBalanced, ProfileUltra, ProfileCustom} {
		if p == "" {
			t.Fatal("profile must be named")
		}
	}
}
