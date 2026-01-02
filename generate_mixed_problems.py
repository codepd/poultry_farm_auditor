#!/usr/bin/env python3
"""
Quick script to generate mixed math problems (addition, subtraction, multiplication, division).
Saves to files for easy printing.
"""

from math_problems_generator import MathProblemGenerator
import sys
import os

def main():
    generator = MathProblemGenerator()
    
    # Default: 20 mixed problems
    num_problems = int(sys.argv[1]) if len(sys.argv) > 1 else 20
    
    # Check if user wants word problems included (default: yes)
    include_words = True
    if len(sys.argv) > 2 and sys.argv[2].lower() in ['n', 'no', 'false']:
        include_words = False
    
    # Check if user wants to save to file (default: yes)
    save_to_file = True
    if len(sys.argv) > 3 and sys.argv[3].lower() in ['n', 'no', 'false', 'print']:
        save_to_file = False
    
    # Optional custom filename
    custom_filename = None
    if len(sys.argv) > 4:
        custom_filename = sys.argv[4]
    
    print(f"\nGenerating {num_problems} mixed math problems...")
    if include_words:
        print("(Including word problems for practice)")
    else:
        print("(Calculation problems only)")
    
    problems = generator.generate_worksheet(
        num_problems=num_problems, 
        include_word_problems=include_words
    )
    
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

