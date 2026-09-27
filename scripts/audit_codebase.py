import os
import re
import json

checks = {
    'DirectiveIR': {
        'keywords': ['DirectiveIR', 'directive_ir', 'DirectiveCompiler'],
        'expected_paths': ['internal/harness/directive', 'internal/directive', 'internal/protocol/directive']
    },
    'NativeAgent': {
        'keywords': ['NativeAgent', 'AgentLoop', 'Step', 'Turn', 'RunLoop'],
        'expected_paths': ['internal/harness/agent', 'internal/runtime']
    },
    'ModelGateway': {
        'keywords': ['ModelGateway', 'ModelRoute', 'QuotaState', 'RouteTarget', 'Fallback', 'AccountPool'],
        'expected_paths': ['internal/harness/gateway', 'internal/modelregistry']
    },
    'BudgetManager': {
        'keywords': ['BudgetManager', 'Envelope', 'HardLimit', 'SoftLimit', 'Reservation', 'blocked_budget'],
        'expected_paths': ['internal/budget', 'internal/harness/runtime']
    },
    'WorkforceRuntime': {
        'keywords': ['Workforce', 'AgentRole', 'AgentBinding', 'LeastWorkforce', 'ResolveWorkforce'],
        'expected_paths': ['internal/harness/team', 'internal/resolver', 'internal/workforcesync']
    },
    'SkillRuntime_and_P0': {
        'keywords': ['SkillPackage', 'NegativeTriggers', 'grounded-implementation', 'implementation-reality-verification', 'surface-protocol-conformance'],
        'expected_paths': ['internal/skillsv3', 'src/prumo/resources/skills']
    },
    'RecipeRuntime': {
        'keywords': ['RecipeDAG', 'Compensation', 'InfiniteRetry', 'StepExecution', 'ValidateRecipe'],
        'expected_paths': ['internal/harness', 'internal/automation', 'internal/protocol/plans']
    },
    'ToolGateway_ACI': {
        'keywords': ['ToolGateway', 'ToolDescriptor', 'SideEffectClass', 'ResultBudget', 'EvidenceHook'],
        'expected_paths': ['internal/toolgateway', 'internal/harness/aci']
    },
    'PermissionEngine': {
        'keywords': ['PermissionEngine', 'PermissionRequest', 'Grant', 'Deny', 'Pending', 'Scope'],
        'expected_paths': ['internal/harness/perm']
    },
    'EvidenceSystem': {
        'keywords': ['EvidenceGraph', 'EvidenceRecord', 'Digest', 'Freshness', 'InvalidationTrigger'],
        'expected_paths': ['internal/protocol/evidence', 'internal/gauntlet']
    },
    'Quality_Gauntlet_FalseGreen': {
        'keywords': ['Gauntlet', 'TotalAssurance', 'FalseGreen', 'BlindSpot', 'DerivedCompletion'],
        'expected_paths': ['internal/gauntlet', 'internal/protocol/gates']
    },
    'PrumoProtocol_Daemon': {
        'keywords': ['PrumoProtocol', 'CapabilityDiscovery', 'VersionNegotiation', 'DaemonServer', 'prumo serve'],
        'expected_paths': ['internal/harness/protocol', 'internal/harness/daemon']
    },
    'CLI_Headless': {
        'keywords': ['prumo ask', 'prumo agent run', 'prumo explain', 'MachineMode', 'StructuredOutput'],
        'expected_paths': ['cmd/prumo']
    },
    'DecisionRuntime': {
        'keywords': ['DecisionRuntime', 'Abstention', 'ReasonCode', 'ClosedDecision'],
        'expected_paths': ['internal/decision', 'internal/planning']
    },
    'GlobalLearning_Experience': {
        'keywords': ['GlobalLearning', 'CandidatePattern', 'ExperienceRecord', 'ProposalReview'],
        'expected_paths': ['internal/experience']
    },
    'ContextCompiler_v2': {
        'keywords': ['ContextManifest', 'ContextCompiler', 'HybridRetrieval', 'RRF', 'TokenBudget'],
        'expected_paths': ['internal/harness/contextv2', 'internal/contextcompiler']
    }
}

audit_results = {}

for domain, cfg in checks.items():
    found_files = []
    found_keywords = {}
    
    # Check expected paths
    for p in cfg['expected_paths']:
        if os.path.exists(p):
            for root, _, files in os.walk(p):
                for f in files:
                    if f.endswith('.go'):
                        found_files.append(os.path.join(root, f))
                        
    # Search keywords in found files and across internal/
    for kw in cfg['keywords']:
        matches = []
        for root, _, files in os.walk('internal'):
            for f in files:
                if f.endswith('.go'):
                    full_path = os.path.join(root, f)
                    try:
                        with open(full_path, 'r', encoding='utf-8', errors='ignore') as fh:
                            content = fh.read()
                        if kw.lower() in content.lower():
                            matches.append(full_path)
                    except Exception:
                        pass
        found_keywords[kw] = matches
        
    audit_results[domain] = {
        'files_count': len(found_files),
        'sample_files': found_files[:5],
        'keywords_matches': {k: len(v) for k, v in found_keywords.items()}
    }

print(json.dumps(audit_results, indent=2))
