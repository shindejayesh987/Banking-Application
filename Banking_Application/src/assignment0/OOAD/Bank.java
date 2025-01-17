package assignment0.OOAD;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.FileNotFoundException;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.Scanner;


public class Bank 
{
	ArrayList<BankAccount> bnk = new ArrayList<BankAccount>();			//A list of BankAccount objects		
	int i = 0;															//This variable will store the line number where 
																		//the current accessed bank account is stored in file
	int flag_login = 0;													//This variable is used to check whether user was successfully able to login
	private int accountNumber;											//User's Bank Account Number (Integer)
	private int pinNumber;												//User's Pin Number (Integer) to access account
	private double savingsBalance;										//User's Savings Balance
	private double checkingBalance;										//User's Checking Balance
	private String acName;												//User's Name
	private boolean check_info = false;									//(boolean) variable to check if user has checking account
	private String fileName = "/Users/saurabh/Documents/workspace"
			+ "/Assignment_0_OOAD/src/assignment0/OOAD/bank_info.csv";	//The string variable in which location of .csv file is stored
	
//	This is a constructor which initializes read_DB() when object of this class is created
	public Bank() throws Exception
	{
		read_DB();
	}
	
//	The Login Method to take input from user for account number and pin number & then this method validates user input with the .csv file 
	public void login() throws Exception
	{
		int attempts = 1;												//Variable to check number of wrongly entered pin number
		i = 0;
		@SuppressWarnings("resource")
		Scanner scan = new Scanner(System.in);							
		System.out.println("Enter the Account Number: ");
		this.accountNumber = scan.nextInt();
		System.out.println("Enter Pin: ");
		this.pinNumber = scan.nextInt();
		Iterator<BankAccount> itr = bnk.iterator();  					//traversing elements of ArrayList object 
		 
		while(itr.hasNext())
		{  
			i++;														//Increment when iterator goes to next element in ArrayList
			BankAccount b = (BankAccount)itr.next();  
		    
			if(b.get_AccNum() == this.accountNumber)					//Checks whether user input and account number from .csv file matches
		    {
		    	while(attempts<3)										//Allows maximum 3 attempts to enter the right passwd 
		    	{
		    		if(b.pin_Check(this.pinNumber))
		    		{
		    			flag_login = 1;
		    			this.savingsBalance = b.get_Savings();
		    			this.acName = b.get_Name();
		    			this.check_info = b.get_Checkinfo();
		    			if(this.check_info)
		    			{
		    				this.checkingBalance = b.get_Checking();
		    			}
		    			
		    			this.menu();									//Menu Method is called
		    			break;
		    		}
		    		
		    		else
		    		{
		    			attempts++;
		    			System.out.println("Re-Enter Pin: ");
		    			this.pinNumber = scan.nextInt();
		    		}		    		
		    	}
		    }
		}
		
		if(flag_login == 0)												//The user has never logged in 
		{
			System.out.println("The Enetered Account Number or Pin Number is Wrong!"
					+ " \nPlease check and try again!!! ");
		}		
	}

//	After successful login user is prompted to choose access of type of account & lets the user to withdraw or deposit amount
	private void menu() throws Exception
	{
		int ch=0;
		@SuppressWarnings("resource")
		Scanner scan = new Scanner(System.in);
		System.out.println("Hello "+ this.acName + 
				", Which Account would you like to access: ");
		System.out.println("1. Checkings");
		System.out.println("2. Savings");
		System.out.println("Choice: ");
		ch = scan.nextInt();
		
		switch(ch)
		{
		case 1: 
				if(this.check_info)
				{
					System.out.println("Your Checkings account "
							+ "has a balance of " + this.checkingBalance);
					double temp = this.dep_WD(this.checkingBalance);	//dep_WD() Method is called			
					this.checkingBalance = temp;
					BankAccount bk = bnk.get(i-1);						
					bk.set_Checking(this.checkingBalance);
					System.out.println("Your New Checkings "
							+ "balance is " + this.checkingBalance);
					System.out.println("Returning to Login Screen...");
				}
				else													//If the user enters checking account even if the user didn't have
				{
					System.out.println("Unsupported Operation!!!");
				}
				
				break;
				
		case 2: System.out.println("Your Savings account "
				+ "has a balance of " + this.savingsBalance);
				double temp = this.dep_WD(this.savingsBalance);			//dep_WD() Method is called	
				this.savingsBalance = temp;
				BankAccount bk = bnk.get(i-1);
				bk.set_Savings(this.savingsBalance);
				System.out.println("Your New Savings "
						+ "balance is " + this.savingsBalance);
				System.out.println("Returning to Login Screen...");
				break;
				
		default: 	System.out.println("Your Savings account "
					+ "has a balance of " + this.savingsBalance);		//If user enters wrong value as option default savings account is selected
					double temp2 = this.dep_WD(this.savingsBalance);	//dep_WD() Method is called	
					this.savingsBalance = temp2;
					BankAccount bk2 = bnk.get(i-1);
					bk2.set_Savings(this.savingsBalance);
					System.out.println("Your New Savings "
							+ "balance is " + this.savingsBalance);
					System.out.println("Returning to Login Screen...");
		}
	}

//	This method accepts the amount from user to deposit or withdraw and add/subtract from the account selected
	private double dep_WD(double account)
	{
		double amount;
		@SuppressWarnings("resource")
		Scanner scan = new Scanner(System.in);
		System.out.println("How would you like to proceed?");
		System.out.println("(Enter in positive number to deposit, "
				+ "negative number to withdraw)");
		System.out.println("Amount: ");
		amount = scan.nextDouble();
		
		if(-(amount)<account)											//Checks if amount withdrawn is less than the total amount present
		{
			account = account + amount;
		}
		else
		{
			System.out.println("Withdrawing amount is greater "
					+ "than the available balance.\n "					//If above condition fails gives this message
					+ "Please Try Again!!!");
		}
		
		return account;
	}

//	This method is called at start by constructor to read all the account information from .csv 
//	and store it in the list of BankAccount objects
	public void read_DB() throws Exception
	{
		String line = null;
		try 
		{
            FileReader fileReader = new FileReader(fileName);
            BufferedReader bufferedReader = new BufferedReader(fileReader);

            while((line = bufferedReader.readLine()) != null) 
            {
            	String[] lines = line.split(",");						//Splits the line where ',' is present and stores it in String Array
            	if (lines.length == 5)
        		{
        			this.check_info = true;								//If the length of the String array is 5 then it has checking account
        			this.accountNumber = Integer.parseInt(lines[0]);
        			this.pinNumber = Integer.parseInt(lines[1]);
        			this.savingsBalance = Double.parseDouble(lines[2]);	//all the account information in that line is stored in the current class variables
        			this.checkingBalance = Double.parseDouble(lines[3]);
        			this.acName = lines[4];
        			BankAccount a = new BankAccount(this.accountNumber, this.check_info);	//new BankAccount object is declared and initialized 
        			a.set_PinNum(this.pinNumber);
        			a.set_Name(this.acName);
        			a.set_Savings(this.savingsBalance);					//all the variables of the declared object are set from this class variables using setters
        			a.set_Checking(this.checkingBalance);
        			bnk.add(a);											//This object is added to bnk ArrayList
        		}
        		else
        		{
        			this.accountNumber = Integer.parseInt(lines[0]);	//If the length of the String array is 4 then it does not have checking account
        			this.pinNumber = Integer.parseInt(lines[1]);
        			this.check_info = false;
        			this.savingsBalance = Double.parseDouble(lines[2]);
        			this.acName = lines[3];
        			this.checkingBalance = 0;
        			BankAccount a = new BankAccount(this.accountNumber, this.check_info);
        			a.set_PinNum(this.pinNumber);
        			a.set_Name(this.acName);
        			a.set_Savings(this.savingsBalance);
        			bnk.add(a);
        		}
            }   
            
            bufferedReader.close();         						
        }
		catch(FileNotFoundException ex) 
		{
            System.out.println(
                "Unable to open file '" + 
                fileName + "'");                
        }
		catch(IOException ex) 
		{
            System.out.println(
                "Error reading file '" 
                + fileName + "'");                  
            // Or we could just do this: 
            // ex.printStackTrace();
        }
	}
	
//	This method is called at the end to confirm change when "Save & Exit" is chosen by user
	public void write_DB() throws Exception
	{
		if(flag_login == 0)												//If not logged in at all. No need to change the .csv file
		{
			System.out.println("Not Logged in Yet.\nTry Again Later!");
		}
		else
		{
			String line = null;											//this variable is used to iterate through the .csv file to read
			String replacer = null;										//this String is set to the values changed after menu is executed 
			String replaced = null;										//this String is the unchanged values in .csv file which should be changed
			BufferedWriter bw = null;									//BufferedWriter & FileWriter class used to write into text files
			FileWriter fw = null;
			BankAccount bk = bnk.get(i-1);								//Object is assigned to account on which the operations are performed
			bk.print_CSV();												//This method prints all the changed values in the account
			try 
			{
				String oldtext = "";									//This String will store all the data which the old.csv file has 
				String newtext = new String();							//This String will store all the data with changed values in .csv file
	            FileReader fileReader = new FileReader(fileName);
	            BufferedReader bufferedReader = new BufferedReader(fileReader);

	            while((line = bufferedReader.readLine()) != null) 
	            {
	            	oldtext += line + "\n";
	            	String[] lines = line.split(",");
	            	if (Integer.parseInt(lines[0]) == this.accountNumber)
	            	{
	            		replaced = line + "\n";
	            		if (this.check_info)							//Check for checking account
	            		{
	            			replacer = Integer.toString(this.accountNumber) + "," + Integer.toString(this.pinNumber) 
	            			+ "," + Double.toString(this.savingsBalance) + "," + Double.toString(this.checkingBalance) 
	            			+ "," + this.acName;

	            		}
	            		else
	            		{
	            			replacer = Integer.toString(this.accountNumber) + "," + Integer.toString(this.pinNumber) 
	            			+ "," + Double.toString(this.savingsBalance)  + "," + this.acName;
	            		}
	            	}
	            }
	            bufferedReader.close();     
	            newtext = oldtext.replaceAll(replaced, replacer);		//Replace the particular string which you want to change
	            
	            fw = new FileWriter(fileName);
				bw = new BufferedWriter(fw);
	            bw.write(newtext);										//Write into the .csv file
	            bw.close();
	        }
			catch(FileNotFoundException ex) 
			{
	            System.out.println(
	                "Unable to open file '" + 
	                fileName + "'");                
	        }
			catch(IOException ex) 
			{
	            System.out.println(
	                "Error reading file '" 
	                + fileName + "'");                  
	            // Or we could just do this: 
	            // ex.printStackTrace();
	        }
		}
	}
		
}
