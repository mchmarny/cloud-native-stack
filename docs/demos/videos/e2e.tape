# VHS Demo: Cloud Native Stack End-to-End Workflow
# Based on docs/demos/e2e.md
#
# Run with: vhs docs/demos/videos/e2e.tape -o docs/demos/videos/e2e.gif
#
# NOTE: Uses `head` to limit long outputs so they display from the top

Output docs/demos/videos/e2e.gif

Set Shell "bash"
Set FontSize 22
Set Width 1400
Set Height 800
Set WindowBar Colorful
Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 10

# Welcome
Type "# Cloud Native Stack: End-to-End Demo"
Enter
Sleep 2s
Type "clear"
Enter

# Step 1: Recipe from CLI
Type "# Recipe from CLI"
Enter
Sleep 1s
Type@60ms "cnsctl recipe \\"
Enter
Type@60ms "  --service eks \\"
Enter
Type@60ms "  --accelerator gb200 \\"
Enter
Type@60ms "  --os ubuntu \\"
Enter
Type@60ms "  --intent training"
Enter
Sleep 500ms
Sleep 4s
Type "clear"
Enter

# Step 2: Recipe from API
Type "# Recipe from API"
Enter
Sleep 1s
Type@60ms 'curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=gb200&intent=training" | jq . | head -50'
Enter
Sleep 4s
Type "clear"
Enter

# Step 3: Make Snapshot (deploy agent)
Type "# Make Snapshot"
Enter
Sleep 1s
Type@60ms "cnsctl snapshot \\"
Enter
Type@60ms "    --deploy-agent \\"
Enter
Type@60ms "    --namespace gpu-operator \\"
Enter
Type@60ms "    --image ghcr.io/mchmarny/cns:latest \\"
Enter
Type@60ms "    --node-selector nodeGroup=customer-gpu"
Enter
Sleep 4s
Type "clear"
Enter

# Step 4: Check Snapshot in ConfigMap
Type "# Check Snapshot in ConfigMap"
Enter
Sleep 1s
Type@60ms "kubectl -n gpu-operator get cm cns-snapshot -o jsonpath='{.data.snapshot\\.yaml}' | yq . | head -40"
Enter
Sleep 4s
Type "clear"
Enter

# Step 5: Recipe from Snapshot
Type "# Recipe from Snapshot"
Enter
Sleep 1s
Type@60ms "cnsctl recipe \\"
Enter
Type@60ms "  --snapshot cm://gpu-operator/cns-snapshot \\"
Enter
Type@60ms "  --intent training \\"
Enter
Type@60ms "  --output recipe.yaml"
Enter
Sleep 3s
Type "clear"
Enter

# Step 6: Recipe Constraints
Type "# Recipe Constraints"
Enter
Sleep 1s
Type "cat recipe.yaml | yq .constraints"
Enter
Sleep 3s
Type "clear"
Enter

# Step 7: Validate Recipe
Type "# Validate Recipe"
Enter
Sleep 1s
Type@60ms "cnsctl validate \\"
Enter
Type@60ms "  --recipe recipe.yaml \\"
Enter
Type@60ms "  --snapshot cm://gpu-operator/cns-snapshot"
Enter
Sleep 4s
Type "clear"
Enter

# Step 8: Bundle from Recipe
Type "# Bundle from Recipe"
Enter
Sleep 1s
Type@60ms "cnsctl bundle \\"
Enter
Type@60ms "  --recipe recipe.yaml \\"
Enter
Type@60ms "  --output ./bundles \\"
Enter
Type@60ms "  --system-node-selector nodeGroup=system-pool \\"
Enter
Type@60ms "  --accelerated-node-selector nodeGroup=customer-gpu \\"
Enter
Type@60ms "  --accelerated-node-toleration nvidia.com/gpu=present:NoSchedule"
Enter
Sleep 4s
Type "clear"
Enter

# Step 9: Check bundle content
Type "# Check bundle content"
Enter
Sleep 1s
Type "tree ./bundles"
Enter
Sleep 4s
Type "clear"
Enter

# Step 10: Bundle from API
Type "# Bundle from Recipe using API"
Enter
Sleep 1s
Type@60ms 'curl -s "https://cns.dgxc.io/v1/recipe?service=eks&accelerator=h100&intent=training" | \'
Enter
Type@60ms '  curl -X POST "https://cns.dgxc.io/v1/bundle?deployer=argocd" \'
Enter
Type@60ms '    -H "Content-Type: application/json" -d @- -o bundles.zip'
Enter
Sleep 2s
Type "unzip -l archive.zip"
Enter
Sleep 3s
Type "clear"
Enter

# Done
Type "# Demo complete! Visit: https://github.com/mchmarny/cloud-native-stack/tree/main"
Enter
Sleep 3s
