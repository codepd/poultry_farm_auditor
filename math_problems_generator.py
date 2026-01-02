#!/usr/bin/env python3
"""
Singapore Math Problem Generator for Third Grade
Generates addition, subtraction, multiplication, and division problems
including word problems (sentence-based) to help with problem-solving skills.
"""

import random
from typing import List, Tuple
from datetime import datetime


class MathProblemGenerator:
    def __init__(self):
        self.problems = []
    
    def generate_addition_problem(self, word_problem: bool = False) -> Tuple[str, int]:
        """Generate an addition problem suitable for third grade."""
        if word_problem:
            scenarios = [
                ("Sarah has {a} stickers. Her friend gives her {b} more stickers. How many stickers does Sarah have now?", "addition"),
                ("There are {a} students in Class 3A and {b} students in Class 3B. How many students are there in both classes?", "addition"),
                ("A library has {a} books on the first shelf and {b} books on the second shelf. How many books are there in total?", "addition"),
                ("Tom collected {a} seashells on Monday and {b} seashells on Tuesday. How many seashells did he collect altogether?", "addition"),
                ("A bakery sold {a} cupcakes in the morning and {b} cupcakes in the afternoon. How many cupcakes were sold in total?", "addition"),
            ]
            scenario, _ = random.choice(scenarios)
            a = random.randint(50, 999)
            b = random.randint(50, 999)
            problem = scenario.format(a=a, b=b)
            answer = a + b
        else:
            a = random.randint(100, 999)
            b = random.randint(100, 999)
            problem = f"{a} + {b} = ?"
            answer = a + b
        
        return problem, answer
    
    def generate_subtraction_problem(self, word_problem: bool = False) -> Tuple[str, int]:
        """Generate a subtraction problem suitable for third grade."""
        if word_problem:
            scenarios = [
                ("Emma had {a} marbles. She gave away {b} marbles to her brother. How many marbles does Emma have left?", "subtraction"),
                ("A store had {a} toys. They sold {b} toys. How many toys are left in the store?", "subtraction"),
                ("There were {a} birds on a tree. {b} birds flew away. How many birds are still on the tree?", "subtraction"),
                ("Lisa saved {a} dollars. She spent {b} dollars on a toy. How much money does she have left?", "subtraction"),
                ("A farmer had {a} eggs. He sold {b} eggs. How many eggs does he have remaining?", "subtraction"),
            ]
            scenario, _ = random.choice(scenarios)
            a = random.randint(200, 999)
            b = random.randint(50, a - 50)  # Ensure positive result
            problem = scenario.format(a=a, b=b)
            answer = a - b
        else:
            a = random.randint(200, 999)
            b = random.randint(50, a - 50)
            problem = f"{a} - {b} = ?"
            answer = a - b
        
        return problem, answer
    
    def generate_multiplication_problem(self, word_problem: bool = False) -> Tuple[str, int]:
        """Generate a multiplication problem suitable for third grade."""
        if word_problem:
            scenarios = [
                ("There are {a} boxes. Each box contains {b} pencils. How many pencils are there in total?", "multiplication"),
                ("A teacher has {a} rows of desks. Each row has {b} desks. How many desks are there altogether?", "multiplication"),
                ("Each bag has {b} apples. If there are {a} bags, how many apples are there in total?", "multiplication"),
                ("A week has 7 days. How many days are there in {a} weeks?", "multiplication"),
                ("Each packet contains {b} cookies. If you buy {a} packets, how many cookies do you have?", "multiplication"),
            ]
            scenario, _ = random.choice(scenarios)
            a = random.randint(2, 12)
            b = random.randint(2, 12)
            problem = scenario.format(a=a, b=b)
            answer = a * b
        else:
            a = random.randint(2, 12)
            b = random.randint(2, 12)
            problem = f"{a} × {b} = ?"
            answer = a * b
        
        return problem, answer
    
    def generate_division_problem(self, with_remainder: bool = False, word_problem: bool = False) -> Tuple[str, str]:
        """Generate a division problem suitable for third grade."""
        if with_remainder:
            # Create problems that will have remainders
            divisor = random.randint(3, 9)
            quotient = random.randint(5, 15)
            remainder = random.randint(1, divisor - 1)
            dividend = divisor * quotient + remainder
            
            if word_problem:
                scenarios = [
                    ("{dividend} candies are shared equally among {divisor} children. How many candies does each child get? How many candies are left over?", "division"),
                    ("A teacher has {dividend} pencils. She wants to put them into {divisor} boxes equally. How many pencils go into each box? How many pencils are left?", "division"),
                    ("{dividend} stickers are divided equally among {divisor} friends. How many stickers does each friend get? How many stickers remain?", "division"),
                    ("There are {dividend} flowers. They are arranged into bouquets of {divisor} flowers each. How many complete bouquets can be made? How many flowers are left?", "division"),
                ]
                scenario, _ = random.choice(scenarios)
                problem = scenario.format(dividend=dividend, divisor=divisor)
                answer = f"{quotient} remainder {remainder}"
            else:
                problem = f"{dividend} ÷ {divisor} = ?"
                answer = f"{quotient} R {remainder}"
        else:
            # Regular division without remainders
            divisor = random.randint(2, 10)
            quotient = random.randint(2, 12)
            dividend = divisor * quotient
            
            if word_problem:
                scenarios = [
                    ("{dividend} apples are shared equally among {divisor} children. How many apples does each child get?", "division"),
                    ("A baker has {dividend} cookies. She wants to put them into {divisor} boxes equally. How many cookies go into each box?", "division"),
                    ("{dividend} books are divided equally among {divisor} shelves. How many books are on each shelf?", "division"),
                    ("There are {dividend} students. They are divided into {divisor} equal groups. How many students are in each group?", "division"),
                    ("{dividend} pencils are shared equally among {divisor} students. How many pencils does each student receive?", "division"),
                ]
                scenario, _ = random.choice(scenarios)
                problem = scenario.format(dividend=dividend, divisor=divisor)
                answer = str(quotient)
            else:
                problem = f"{dividend} ÷ {divisor} = ?"
                answer = str(quotient)
        
        return problem, answer
    
    def generate_worksheet(self, num_problems: int = 20, include_word_problems: bool = True):
        """Generate a complete worksheet with mixed problems."""
        problems = []
        
        # Calculate distribution
        problems_per_type = num_problems // 4
        remainder = num_problems % 4
        
        # Generate problems
        # Addition
        for i in range(problems_per_type + (1 if remainder > 0 else 0)):
            word_prob = include_word_problems and random.random() < 0.4  # 40% word problems
            prob, ans = self.generate_addition_problem(word_problem=word_prob)
            problems.append(("Addition", prob, ans))
        
        # Subtraction
        for i in range(problems_per_type + (1 if remainder > 1 else 0)):
            word_prob = include_word_problems and random.random() < 0.4
            prob, ans = self.generate_subtraction_problem(word_problem=word_prob)
            problems.append(("Subtraction", prob, ans))
        
        # Multiplication
        for i in range(problems_per_type + (1 if remainder > 2 else 0)):
            word_prob = include_word_problems and random.random() < 0.4
            prob, ans = self.generate_multiplication_problem(word_problem=word_prob)
            problems.append(("Multiplication", prob, ans))
        
        # Division (mix of remainder and regular)
        for i in range(problems_per_type):
            word_prob = include_word_problems and random.random() < 0.4
            with_remainder = random.random() < 0.5  # 50% with remainders
            prob, ans = self.generate_division_problem(with_remainder=with_remainder, word_problem=word_prob)
            problems.append(("Division", prob, ans))
        
        random.shuffle(problems)
        return problems
    
    def print_worksheet(self, problems: List[Tuple[str, str, str]], show_answers: bool = False):
        """Print the worksheet in a nice format."""
        print("\n" + "="*70)
        print("SINGAPORE MATH PRACTICE WORKSHEET - THIRD GRADE")
        print(f"Generated on: {datetime.now().strftime('%B %d, %Y')}")
        print("="*70 + "\n")
        
        for i, (operation, problem, answer) in enumerate(problems, 1):
            print(f"{i}. [{operation}]")
            print(f"   {problem}")
            if show_answers:
                print(f"   Answer: {answer}")
            print()
        
        if not show_answers:
            print("\n" + "="*70)
            print("ANSWER KEY")
            print("="*70 + "\n")
            for i, (operation, problem, answer) in enumerate(problems, 1):
                print(f"{i}. {answer}")
            print()
    
    def save_worksheet_to_file(self, problems: List[Tuple[str, str, str]], filename: str = None, separate_answer_key: bool = True):
        """Save the worksheet to a file. Creates two files: one with problems, one with answers."""
        if filename is None:
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
            filename = f"math_worksheet_{timestamp}"
        
        # Remove extension if provided
        if filename.endswith('.txt'):
            filename = filename[:-4]
        
        # Create problems file
        problems_file = f"{filename}.txt"
        answer_file = f"{filename}_answers.txt"
        
        with open(problems_file, 'w', encoding='utf-8') as f:
            f.write("="*70 + "\n")
            f.write("SINGAPORE MATH PRACTICE WORKSHEET - THIRD GRADE\n")
            f.write(f"Generated on: {datetime.now().strftime('%B %d, %Y')}\n")
            f.write("="*70 + "\n\n")
            f.write("Name: ___________________________  Date: ___________\n\n")
            
            for i, (operation, problem, answer) in enumerate(problems, 1):
                f.write(f"{i}. [{operation}]\n")
                f.write(f"   {problem}\n")
                f.write("\n")
        
        # Create answer key file
        with open(answer_file, 'w', encoding='utf-8') as f:
            f.write("="*70 + "\n")
            f.write("ANSWER KEY\n")
            f.write(f"Generated on: {datetime.now().strftime('%B %d, %Y')}\n")
            f.write("="*70 + "\n\n")
            
            for i, (operation, problem, answer) in enumerate(problems, 1):
                f.write(f"{i}. {answer}\n")
        
        return problems_file, answer_file


def main():
    generator = MathProblemGenerator()
    
    print("Singapore Math Problem Generator for Third Grade")
    print("-" * 50)
    print("\nOptions:")
    print("1. Generate a worksheet with mixed problems")
    print("2. Generate addition problems only")
    print("3. Generate subtraction problems only")
    print("4. Generate multiplication problems only")
    print("5. Generate division problems only (with and without remainders)")
    print("6. Generate word problems only (all operations)")
    
    choice = input("\nEnter your choice (1-6): ").strip()
    
    if choice == "1":
        num = int(input("How many problems? (default 20): ") or "20")
        include_words = input("Include word problems? (y/n, default y): ").strip().lower() != 'n'
        problems = generator.generate_worksheet(num_problems=num, include_word_problems=include_words)
        generator.print_worksheet(problems)
    
    elif choice == "2":
        num = int(input("How many problems? (default 10): ") or "10")
        word_only = input("Word problems only? (y/n, default n): ").strip().lower() == 'y'
        problems = []
        for _ in range(num):
            prob, ans = generator.generate_addition_problem(word_problem=word_only)
            problems.append(("Addition", prob, ans))
        generator.print_worksheet(problems)
    
    elif choice == "3":
        num = int(input("How many problems? (default 10): ") or "10")
        word_only = input("Word problems only? (y/n, default n): ").strip().lower() == 'y'
        problems = []
        for _ in range(num):
            prob, ans = generator.generate_subtraction_problem(word_problem=word_only)
            problems.append(("Subtraction", prob, ans))
        generator.print_worksheet(problems)
    
    elif choice == "4":
        num = int(input("How many problems? (default 10): ") or "10")
        word_only = input("Word problems only? (y/n, default n): ").strip().lower() == 'y'
        problems = []
        for _ in range(num):
            prob, ans = generator.generate_multiplication_problem(word_problem=word_only)
            problems.append(("Multiplication", prob, ans))
        generator.print_worksheet(problems)
    
    elif choice == "5":
        num = int(input("How many problems? (default 10): ") or "10")
        word_only = input("Word problems only? (y/n, default n): ").strip().lower() == 'y'
        with_remainder_only = input("Remainder problems only? (y/n, default n): ").strip().lower() == 'y'
        problems = []
        for _ in range(num):
            with_remainder = with_remainder_only or (not with_remainder_only and random.random() < 0.5)
            prob, ans = generator.generate_division_problem(with_remainder=with_remainder, word_problem=word_only)
            problems.append(("Division", prob, ans))
        generator.print_worksheet(problems)
    
    elif choice == "6":
        num = int(input("How many word problems? (default 15): ") or "15")
        problems = []
        operations = ['add', 'sub', 'mul', 'div'] * (num // 4 + 1)
        random.shuffle(operations)
        for op in operations[:num]:
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
        generator.print_worksheet(problems)
    
    else:
        print("Invalid choice. Generating default worksheet...")
        problems = generator.generate_worksheet()
        generator.print_worksheet(problems)


if __name__ == "__main__":
    main()

