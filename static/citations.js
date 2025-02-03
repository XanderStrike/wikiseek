document.addEventListener('DOMContentLoaded', function() {
    // Add event listeners to all citation markers
    document.addEventListener('mouseover', function(event) {
        if (event.target.classList.contains('citation-marker')) {
            const table = event.target.querySelector('.citation-table');
            if (table) {
                table.style.display = 'block';
                updatePosition(event, table);
            }
        }
    });

    document.addEventListener('mouseout', function(event) {
        if (event.target.classList.contains('citation-marker')) {
            const table = event.target.querySelector('.citation-table');
            if (table) {
                table.style.display = 'none';
            }
        }
    });

    document.addEventListener('mousemove', function(event) {
        const activeMarker = event.target.closest('.citation-marker');
        if (activeMarker) {
            const table = activeMarker.querySelector('.citation-table');
            if (table && table.style.display === 'block') {
                updatePosition(event, table);
            }
        }
    });
});

function updatePosition(event, table) {
    const padding = 10;
    const rect = table.getBoundingClientRect();
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    
    // Calculate position relative to viewport
    let left = event.clientX + padding;
    let top = event.clientY + padding;
    
    // Adjust if would overflow right edge
    if (left + rect.width > viewportWidth) {
        left = event.clientX - rect.width - padding;
    }
    
    // Adjust if would overflow bottom edge
    if (top + rect.height > viewportHeight) {
        top = event.clientY - rect.height - padding;
    }
    
    // Use fixed positioning with viewport coordinates
    table.style.position = 'fixed';
    table.style.left = left + 'px';
    table.style.top = top + 'px';
}
