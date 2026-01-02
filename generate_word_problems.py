#!/usr/bin/env python3
"""
Quick script to generate word problems for practice.
Focuses on sentence-based problems to help with problem-solving skills.
Saves to files for easy printing.
"""

from math_problems_generator import MathProblemGenerator
import sys
import random

def main():
    generator = MathProblemGenerator()
    
    # Default: 15 word problems with mixed operations
    num_problems = int(sys.argv[1]) if len(sys.argv) > 1 else 15
    
    # Check if user wants to save to file (default: yes)
    save_to_file = True
    if len(sys.argv) > 2 and sys.argv[2].lower() in ['n', 'no', 'false', 'print']:
        save_to_file = False
    
    # Optional custom filename
    custom_filename = None
    if len(sys.argv) > 3:
        custom_filename = sys.argv[3]
    
    print(f"\nGenerating {num_problems} word problems for practice...")
    
    problems = []
    operations = ['add', 'sub', 'mul', 'div'] * (num_problems // 4 + 1)
    random.shuffle(operations)
    
    for op in operations[:num_problems]:
        if op == 'add':
            prob, ans = generator.generate_addition_problem(word_problem=True)
            problems.append(("Addition", prob, ans))
        elif op == 'sub':
            prob, ans = generator.generate_subtraction_problem(word_problem=True)
            problems.append(("Subtraction", prob, ans))
        elif op == 'mul':
            prob, ans = generator.generate_multiplication_problem(word_problem=True)
            problems.append(("Multiplication", prob, ans))
        else:
            with_remainder = random.random() < 0.5
            prob, ans = generator.generate_division_problem(with_remainder=with_remainder, word_problem=True)
            problems.append(("Division", prob, ans))
    
    if save_to_file:
        problems_file, answer_file = generator.save_worksheet_to_file(
            problems, 
            filename=custom_filename
        )
        print(f"\n✓ Worksheet saved to: {problems_file}")
        print(f"✓ Answer key saved to: {answer_file}")
        print(f"\nFiles are ready to print!")
    else:
        generator.print_worksheet(problems)

if __name__ == "__main__":
    main()

