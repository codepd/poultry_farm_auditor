"""
Egg item name normalization utilities.
Handles all egg types including LARGE, MEDIUM, SMALL, BROKEN, DOUBLE_YOLK, and DIRT eggs.
"""

def normalize_egg_item_name(item_name):
    """
    Normalize egg item names for consistency.
    Maps various names to standard egg types.
    
    Returns: (normalized_name, egg_type)
    """
    if not item_name:
        return item_name, None
    
    item_upper = str(item_name).upper().strip()
    
    # Large eggs
    if any(keyword in item_upper for keyword in ['LARGE', 'CORRECT EGG', 'EXPORT EGG', 'CORRECT SIZE']):
        return 'LARGE EGG', 'LARGE'
    
    # Medium eggs
    if 'MEDIUM' in item_upper:
        return 'MEDIUM EGG', 'MEDIUM'
    
    # Small eggs
    if 'SMALL' in item_upper:
        return 'SMALL EGG', 'SMALL'
    
    # Broken eggs
    if any(keyword in item_upper for keyword in ['BROKEN', 'BREAK']):
        return 'BROKEN EGG', 'BROKEN'
    
    # Double yolk eggs
    if any(keyword in item_upper for keyword in ['DOUBLE', 'YOLK', 'DOUBLE YOLK']):
        return 'DOUBLE YOLK EGG', 'DOUBLE_YOLK'
    
    # Dirt eggs
    if any(keyword in item_upper for keyword in ['DIRT', 'SOIL', 'DIRTY']):
        return 'DIRT EGG', 'DIRT'
    
    # Default to large if it's clearly an egg but type is unclear
    if 'EGG' in item_upper:
        return 'LARGE EGG', 'LARGE'
    
    return item_name, None

def get_egg_type(item_name):
    """Extract egg type from item name."""
    _, egg_type = normalize_egg_item_name(item_name)
    return egg_type

def is_egg_item(item_name):
    """Check if an item is an egg."""
    if not item_name:
        return False
    item_upper = str(item_name).upper().strip()
    return 'EGG' in item_upper or any(keyword in item_upper for keyword in [
        'LARGE', 'MEDIUM', 'SMALL', 'CORRECT', 'EXPORT', 'BROKEN', 'BREAK',
        'DOUBLE', 'YOLK', 'DIRT', 'SOIL', 'DIRTY'
    ])

