// Initialize visualization components
let network = null;
let nodes = new vis.DataSet([]);
let edges = new vis.DataSet([]);
let stepHistory = [];
let metrics = {
    activeAgents: 0,
    totalSteps: 0,
    cyclesDetected: 0
};

// Network visualization options
const options = {
    nodes: {
        shape: 'dot',
        size: 30,
        font: {
            size: 14
        },
        borderWidth: 2,
        shadow: true
    },
    edges: {
        width: 2,
        arrows: {
            to: {
                enabled: true,
                scaleFactor: 1
            }
        },
        shadow: true
    },
    physics: {
        stabilization: false,
        barnesHut: {
            gravitationalConstant: -80000,
            springConstant: 0.001,
            springLength: 200
        }
    }
};

// Initialize the visualization
function initVisualization() {
    const container = document.getElementById('workflow-graph');
    network = new vis.Network(container, { nodes, edges }, options);

    // Network events
    network.on('click', function(params) {
        if (params.nodes.length > 0) {
            const nodeId = params.nodes[0];
            highlightConnections(nodeId);
        }
    });

    // Initialize WebSocket connection
    connectWebSocket();
}

// WebSocket connection
function connectWebSocket() {
    const ws = new WebSocket('ws://' + window.location.host + '/ws');

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);
        handleEvent(data);
    };

    ws.onclose = function() {
        setTimeout(connectWebSocket, 1000); // Reconnect after 1 second
    };
}

// Handle incoming events
function handleEvent(event) {
    switch(event.type) {
        case 'workflow_started':
            initializeWorkflow(event.data);
            break;
        case 'agent_started':
            handleAgentStarted(event.data);
            break;
        case 'agent_completed':
            handleAgentCompleted(event.data);
            break;
        case 'message_sent':
            handleMessageSent(event.data);
            break;
        case 'cycle_detected':
            handleCycleDetected(event.data);
            break;
        case 'workflow_ended':
            handleWorkflowEnded(event.data);
            break;
    }
    updateMetrics();
}

// Initialize workflow visualization
function initializeWorkflow(data) {
    nodes.clear();
    edges.clear();
    stepHistory = [];
    metrics = {
        activeAgents: 0,
        totalSteps: 0,
        cyclesDetected: 0
    };

    // Add agents as nodes
    data.agents.forEach(agent => {
        nodes.add({
            id: agent,
            label: agent,
            color: {
                background: '#97C2FC',
                border: '#2B7CE9'
            }
        });
    });

    // Add connections as edges
    for (const [from, toList] of Object.entries(data.connections)) {
        toList.forEach(to => {
            edges.add({
                from: from,
                to: to,
                id: `${from}-${to}`
            });
        });
    }

    // Color team leaders differently
    for (const [team, leader] of Object.entries(data.teamLeaders)) {
        nodes.update({
            id: leader,
            color: {
                background: '#FFA500',
                border: '#FF8C00'
            }
        });
    }
}

// Handle agent started event
function handleAgentStarted(data) {
    metrics.activeAgents++;
    metrics.totalSteps++;

    nodes.update({
        id: data.agent_name,
        color: {
            background: '#7BE141',
            border: '#4CAF50'
        }
    });

    addStepToHistory({
        step: data.step,
        agent: data.agent_name,
        status: 'active',
        timestamp: new Date().toLocaleTimeString()
    });
}

// Handle agent completed event
function handleAgentCompleted(data) {
    metrics.activeAgents--;

    nodes.update({
        id: data.agent_name,
        color: {
            background: '#97C2FC',
            border: '#2B7CE9'
        }
    });

    updateStepInHistory(data.step, 'completed');
}

// Handle message sent event
function handleMessageSent(data) {
    // Highlight the edge temporarily
    const edgeId = `${data.from_agent}-${data.to_agent}`;
    edges.update({
        id: edgeId,
        color: {
            color: '#FF0000',
            highlight: '#FF0000'
        }
    });

    // Reset edge color after animation
    setTimeout(() => {
        edges.update({
            id: edgeId,
            color: null
        });
    }, 1000);
}

// Handle cycle detected event
function handleCycleDetected(data) {
    metrics.cyclesDetected++;

    const edgeId = `${data.from_agent}-${data.to_agent}`;
    edges.update({
        id: edgeId,
        color: {
            color: '#FF0000',
            highlight: '#FF0000'
        },
        dashes: true
    });

    addStepToHistory({
        step: -1,
        agent: `${data.from_agent} → ${data.to_agent}`,
        status: 'cycle',
        timestamp: new Date().toLocaleTimeString(),
        count: data.count
    });
}

// Handle workflow ended event
function handleWorkflowEnded(data) {
    // Reset all node colors
    nodes.forEach(node => {
        nodes.update({
            id: node.id,
            color: {
                background: '#97C2FC',
                border: '#2B7CE9'
            }
        });
    });

    addStepToHistory({
        step: -1,
        agent: 'Workflow',
        status: 'completed',
        timestamp: new Date().toLocaleTimeString()
    });
}

// Add step to history panel
function addStepToHistory(step) {
    const historyPanel = document.getElementById('step-history');
    const stepElement = document.createElement('div');
    stepElement.className = `step-item ${step.status}`;
    stepElement.innerHTML = `
        <div class="step-header">
            <span class="step-number">${step.step > 0 ? `Step ${step.step}` : ''}</span>
            <span class="step-time">${step.timestamp}</span>
        </div>
        <div class="step-content">
            <span class="step-agent">${step.agent}</span>
            <span class="step-status">${step.status}</span>
            ${step.count ? `<span class="step-cycle-count">(Cycle #${step.count})</span>` : ''}
        </div>
    `;
    historyPanel.insertBefore(stepElement, historyPanel.firstChild);
    stepHistory.push(step);
}

// Update step in history panel
function updateStepInHistory(stepNumber, status) {
    const historyPanel = document.getElementById('step-history');
    const stepElements = historyPanel.getElementsByClassName('step-item');
    for (let element of stepElements) {
        if (element.querySelector('.step-number').textContent === `Step ${stepNumber}`) {
            element.className = `step-item ${status}`;
            element.querySelector('.step-status').textContent = status;
            break;
        }
    }
}

// Update metrics panel
function updateMetrics() {
    document.getElementById('active-agents').textContent = metrics.activeAgents;
    document.getElementById('total-steps').textContent = metrics.totalSteps;
    document.getElementById('cycles-detected').textContent = metrics.cyclesDetected;
}

// Highlight connections for selected node
function highlightConnections(nodeId) {
    const connectedEdges = network.getConnectedEdges(nodeId);
    const connectedNodes = network.getConnectedNodes(nodeId);

    // Reset all nodes and edges
    nodes.forEach(node => {
        nodes.update({
            id: node.id,
            opacity: 0.3
        });
    });
    edges.forEach(edge => {
        edges.update({
            id: edge.id,
            opacity: 0.3
        });
    });

    // Highlight selected node and its connections
    nodes.update({
        id: nodeId,
        opacity: 1
    });
    connectedNodes.forEach(node => {
        nodes.update({
            id: node,
            opacity: 1
        });
    });
    connectedEdges.forEach(edge => {
        edges.update({
            id: edge,
            opacity: 1
        });
    });
}

// Export workflow diagram
document.getElementById('exportBtn').addEventListener('click', function() {
    const canvas = network.canvas.frame.canvas;
    const link = document.createElement('a');
    link.download = 'workflow-diagram.png';
    link.href = canvas.toDataURL();
    link.click();
});

// Clear history
document.getElementById('clearBtn').addEventListener('click', function() {
    document.getElementById('step-history').innerHTML = '';
    stepHistory = [];
    metrics = {
        activeAgents: 0,
        totalSteps: 0,
        cyclesDetected: 0
    };
    updateMetrics();
});

// Initialize visualization when page loads
window.addEventListener('load', initVisualization);
