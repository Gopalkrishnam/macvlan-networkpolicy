package controllers

import (
	//"fmt"
	"time"

	mvlanv1 "github.com/k8snetworkplumbingwg/macvlan-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1"
	mvlanfake "github.com/k8snetworkplumbingwg/macvlan-networkpolicy/pkg/client/clientset/versioned/fake"
	mvlaninformerv1 "github.com/k8snetworkplumbingwg/macvlan-networkpolicy/pkg/client/informers/externalversions"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeNetworkPolicyConfigStub struct {
	CounterAdd    int
	CounterUpdate int
	CounterDelete int
	CounterSynced int
}

func (f *FakeNetworkPolicyConfigStub) OnPolicyAdd(_ *mvlanv1.MacvlanNetworkPolicy) {
	f.CounterAdd++
}

func (f *FakeNetworkPolicyConfigStub) OnPolicyUpdate(_, _ *mvlanv1.MacvlanNetworkPolicy) {
	f.CounterUpdate++
}

func (f *FakeNetworkPolicyConfigStub) OnPolicyDelete(_ *mvlanv1.MacvlanNetworkPolicy) {
	f.CounterDelete++
}

func (f *FakeNetworkPolicyConfigStub) OnPolicySynced() {
	f.CounterSynced++
}

func NewFakeNetworkPolicyConfig(stub *FakeNetworkPolicyConfigStub) *NetworkPolicyConfig {
	configSync := 15 * time.Minute
	fakeClient := mvlanfake.NewSimpleClientset()
	informerFactory := mvlaninformerv1.NewSharedInformerFactoryWithOptions(fakeClient, configSync)
	policyConfig := NewNetworkPolicyConfig(informerFactory.K8sCniCncfIo().V1().MacvlanNetworkPolicies(), configSync)
	policyConfig.RegisterEventHandler(stub)
	return policyConfig
}

func NewNetworkPolicy(namespace, name string) *mvlanv1.MacvlanNetworkPolicy {
	return &mvlanv1.MacvlanNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

var _ = Describe("networkpolicy config", func() {
	It("check add handler", func() {
		stub := &FakeNetworkPolicyConfigStub{}
		networkPolicyConfig := NewFakeNetworkPolicyConfig(stub)
		networkPolicyConfig.handleAddPolicy(NewNetworkPolicy("testns1", "test1"))
		Expect(stub.CounterAdd).To(Equal(1))
		Expect(stub.CounterUpdate).To(Equal(0))
		Expect(stub.CounterDelete).To(Equal(0))
		Expect(stub.CounterSynced).To(Equal(0))
	})

	It("check update handler", func() {
		stub := &FakeNetworkPolicyConfigStub{}
		networkPolicyConfig := NewFakeNetworkPolicyConfig(stub)
		networkPolicyConfig.handleUpdatePolicy(
			NewNetworkPolicy("testns1", "test1"),
			NewNetworkPolicy("testns2", "test1"))
		Expect(stub.CounterAdd).To(Equal(0))
		Expect(stub.CounterUpdate).To(Equal(1))
		Expect(stub.CounterDelete).To(Equal(0))
		Expect(stub.CounterSynced).To(Equal(0))
	})

	It("check delete handler", func() {
		stub := &FakeNetworkPolicyConfigStub{}
		networkPolicyConfig := NewFakeNetworkPolicyConfig(stub)
		networkPolicyConfig.handleDeletePolicy(NewNetworkPolicy("testns1", "test1"))
		Expect(stub.CounterAdd).To(Equal(0))
		Expect(stub.CounterUpdate).To(Equal(0))
		Expect(stub.CounterDelete).To(Equal(1))
		Expect(stub.CounterSynced).To(Equal(0))
	})
})

var _ = Describe("networkpolicy controller", func() {
	It("Initialize and verify empty", func() {
		policyChanges := NewPolicyChangeTracker()
		policyMap := make(PolicyMap)
		policyMap.Update(policyChanges)
		Expect(len(policyMap)).To(Equal(0))
	})

	It("Add policy and verify", func() {
		policyChanges := NewPolicyChangeTracker()
		policyChanges.Update(nil, NewNetworkPolicy("testns1", "test1"))
		policyChanges.Update(nil, NewNetworkPolicy("testns2", "test2"))

		policyMap := make(PolicyMap)
		policyMap.Update(policyChanges)
		Expect(len(policyMap)).To(Equal(2))

		policyTest1, ok := policyMap[types.NamespacedName{Namespace: "testns1", Name: "test1"}]
		Expect(ok).To(BeTrue())
		Expect(policyTest1.Name()).To(Equal("test1"))
		Expect(policyTest1.Namespace()).To(Equal("testns1"))
		policyTest2, ok := policyMap[types.NamespacedName{Namespace: "testns2", Name: "test2"}]
		Expect(ok).To(BeTrue())
		Expect(policyTest2.Name()).To(Equal("test2"))
		Expect(policyTest2.Namespace()).To(Equal("testns2"))
	})

	It("Add policy then delete it and verify", func() {
		policyChanges := NewPolicyChangeTracker()
		policyChanges.Update(nil, NewNetworkPolicy("testns1", "test1"))
		policyChanges.Update(nil, NewNetworkPolicy("testns2", "test2"))
		policyChanges.Update(NewNetworkPolicy("testns1", "test1"), nil)

		policyMap := make(PolicyMap)
		policyMap.Update(policyChanges)
		Expect(len(policyMap)).To(Equal(1))

		policyTest2, ok := policyMap[types.NamespacedName{Namespace: "testns2", Name: "test2"}]
		Expect(ok).To(BeTrue())
		Expect(policyTest2.Name()).To(Equal("test2"))
		Expect(policyTest2.Namespace()).To(Equal("testns2"))
	})

	It("invalid Update case", func() {
		policyChanges := NewPolicyChangeTracker()
		Expect(policyChanges.Update(nil, nil)).To(BeFalse())
	})

	It("Add policy then update it and verify", func() {
		policyChanges := NewPolicyChangeTracker()
		policyChanges.Update(nil, NewNetworkPolicy("testns1", "test1"))
		policyChanges.Update(
			NewNetworkPolicy("testns1", "test1"),
			NewNetworkPolicy("testns1", "test1"))

		policyMap := make(PolicyMap)
		policyMap.Update(policyChanges)
		Expect(len(policyMap)).To(Equal(1))

		policyTest1, ok := policyMap[types.NamespacedName{Namespace: "testns1", Name: "test1"}]
		Expect(ok).To(BeTrue())
		Expect(policyTest1.Name()).To(Equal("test1"))
		Expect(policyTest1.Namespace()).To(Equal("testns1"))
	})
})
