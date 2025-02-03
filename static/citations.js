document.addEventListener('DOMContentLoaded', function() {
    let pinnedCitation = null;

    // Handle clicks on citation markers
    document.addEventListener('click', function(event) {
        const marker = event.target.closest('.citation-marker');
        
        // If clicking outside any citation marker, unpin current citation
        if (!marker) {
            if (pinnedCitation) {
                pinnedCitation.classList.remove('pinned');
                const table = pinnedCitation.querySelector('.citation-table');
                if (table) {
                    table.style.display = 'none';
                }
                pinnedCitation = null;
            }
            return;
        }

        // Handle click on citation marker
        if (marker) {
            event.preventDefault();
            event.stopPropagation();
            
            // If clicking the currently pinned citation, unpin it
            if (pinnedCitation === marker) {
                marker.classList.remove('pinned');
                const table = marker.querySelector('.citation-table');
                if (table) {
                    table.style.display = 'none';
                }
                pinnedCitation = null;
                return;
            }

            // Unpin previous citation if exists
            if (pinnedCitation) {
                pinnedCitation.classList.remove('pinned');
                const oldTable = pinnedCitation.querySelector('.citation-table');
                if (oldTable) {
                    oldTable.style.display = 'none';
                }
            }

            // Pin new citation
            marker.classList.add('pinned');
            const table = marker.querySelector('.citation-table');
            if (table) {
                table.style.display = 'block';
                updatePosition(event, table);
            }
            pinnedCitation = marker;
        }
    });

    // Show/hide on hover only when not pinned
    document.addEventListener('mouseover', function(event) {
        const marker = event.target.closest('.citation-marker');
        if (marker && !marker.classList.contains('pinned')) {
            const table = marker.querySelector('.citation-table');
            if (table) {
                table.style.display = 'block';
                updatePosition(event, table);
            }
        }
    });

    document.addEventListener('mouseout', function(event) {
        const marker = event.target.closest('.citation-marker');
        if (marker && !marker.classList.contains('pinned')) {
            const table = marker.querySelector('.citation-table');
            if (table) {
                table.style.display = 'none';
            }
        }
    });

    document.addEventListener('mousemove', function(event) {
        const marker = event.target.closest('.citation-marker');
        if (marker && !marker.classList.contains('pinned')) {
            const table = marker.querySelector('.citation-table');
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
