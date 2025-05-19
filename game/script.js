// Shape definitions using clip-path coordinates
const shapes = {
    circle: 'circle(50%)',
    square: 'polygon(0% 0%, 100% 0%, 100% 100%, 0% 100%)',
    pentagon: 'polygon(50% 0%, 100% 38%, 82% 100%, 18% 100%, 0% 38%)',
    hexagon: 'polygon(25% 0%, 75% 0%, 100% 50%, 75% 100%, 25% 100%, 0% 50%)',
    trapezoid: 'polygon(20% 0%, 80% 0%, 100% 100%, 0% 100%)'
};

// Convert clip-path string to points array for animation
function parseClipPath(clipPath) {
    if (clipPath.startsWith('circle')) {
        // For circle, create points in a circular pattern
        const points = [];
        const steps = 36;
        for (let i = 0; i < steps; i++) {
            const angle = (i / steps) * Math.PI * 2;
            points.push([
                50 + 50 * Math.cos(angle),
                50 + 50 * Math.sin(angle)
            ]);
        }
        return points;
    }
    
    return clipPath
        .match(/polygon\((.*?)\)/)[1]
        .split(', ')
        .map(coord => coord.split(' ').map(val => 
            parseFloat(val.replace('%', ''))
        ));
}

// Main shapeshift function
function shapeshift() {
    const shape = document.getElementById('shapeSelect').value;
    const color = document.getElementById('colorPicker').value;
    const size = document.getElementById('sizeSlider').value;
    const element = document.getElementById('morphingShape');
    
    // Get current and target shape points
    const currentClipPath = getComputedStyle(element).clipPath || shapes.circle;
    const currentPoints = parseClipPath(currentClipPath);
    const targetPoints = parseClipPath(shapes[shape]);
    
    // Ensure both point arrays have same length
    while (currentPoints.length < targetPoints.length) {
        currentPoints.push(currentPoints[currentPoints.length - 1]);
    }
    while (targetPoints.length < currentPoints.length) {
        targetPoints.push(targetPoints[targetPoints.length - 1]);
    }
    
    // Create animation timeline
    const timeline = anime.timeline({
        duration: 800,
        easing: 'cubicBezier(.5, .05, .1, .3)',
        update: function(anim) {
            // Calculate current points
            const currentValues = currentPoints.map((point, i) => {
                const targetPoint = targetPoints[i];
                return [
                    point[0] + (targetPoint[0] - point[0]) * (anim.progress / 100),
                    point[1] + (targetPoint[1] - point[1]) * (anim.progress / 100)
                ];
            });
            
            // Apply current shape
            const clipPath = `polygon(${currentValues
                .map(point => `${point[0]}% ${point[1]}%`)
                .join(', ')})`;
            element.style.clipPath = clipPath;
        }
    });
    
    // Add animations to timeline
    timeline
        .add({
            targets: element,
            backgroundColor: color,
            duration: 800
        })
        .add({
            targets: '.shape-container',
            width: size + 'px',
            height: size + 'px',
            duration: 800
        }, '-=800');

    // Update HUD
    document.getElementById('hudShape').textContent = shape;
    document.getElementById('hudColor').textContent = color;
    document.getElementById('hudSize').textContent = Math.round((size / 200) * 100);
}

// Initialize with default blue circle
document.addEventListener('DOMContentLoaded', () => {
    const element = document.getElementById('morphingShape');
    element.style.clipPath = shapes.circle;
});