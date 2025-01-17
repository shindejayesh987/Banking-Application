package assignment0.OOAD;

public class BankAccount 
{
	
	private int accountNumber;					//User's Bank Account Number (Integer)
	private int pinNumber;						//User's Pin Number (Integer) to access account
	private double savingsBalance;				//User's Savings Balance
	private double checkingBalance;				//User's Checking Balance
	private String acName;						//User's Name
	private boolean check_info = false;			//(boolean) variable to check if user has checking account
	
//	Constructer is overloaded with 2 parameters of input
//	These are Account Number and Boolean Number which are assigned to the class variables
	public BankAccount(int accountNumber, boolean check_info)
	{
		this.accountNumber = accountNumber;
		this.check_info = check_info;
	}

//	getter to return Savings Balance 
	public double get_Savings()
	{
		return this.savingsBalance;
	}
	
//	getter to return boolean variable todetermine if this user has checking account
	public boolean get_Checkinfo()
	{
		return this.check_info;
	}
	
//	getter to return Checking Balance
	public double get_Checking()
	{
		if(this.check_info)
		{
			return this.checkingBalance;
		}
		else
		{
			System.out.println("UnsupportedOperationException");
			return (0);
		}
	}
	
//	getter to return the name of User
	public String get_Name()
	{
		return this.acName;
	}
	
//	getter to return Account Number
	public int get_AccNum()
	{
		return this.accountNumber;
	}
	
//	setter to set pin number of this class variable
	public void set_PinNum(int pinNumber)
	{
		this.pinNumber = pinNumber;
	}
	
//	setter to set user name of this class variable
	public void set_Name(String acName)
	{
		this.acName = acName;
	}
	
//	setter to set Saving Balance of this class variable
	public void set_Savings(double savingsBalance) throws Exception
	{
		this.savingsBalance = savingsBalance;
	}
	
//	setter to set Checking Balance of this class variable
	public void set_Checking(double checkingBalance) throws Exception
	{
		if (this.check_info)
		{
			this.checkingBalance = checkingBalance;
		}
		else
		{
			System.out.println("UnsupportedOperationException");
		}
		
	}
	
//	This Fn returns boolean value to check whether passed parameter pin matches with class variable pinNumber
	public boolean pin_Check(int pinNumber) throws Exception
	{
		if (this.pinNumber == pinNumber)
		{
			return true;
		}
		else
		{
			return false;
		}
	}
	
//	This Method prints the current values of the class variables
	public void print_CSV()
	{
		if (this.check_info)
		{
			System.out.println("The changed information is: ");
			System.out.println("- Account Number = "+ this.accountNumber);
			System.out.println("- Pin Number = "+ this.pinNumber);
			System.out.println("- Savings Balance = "+ this.savingsBalance);
			System.out.println("- Checking Balance = "+ this.checkingBalance);
			System.out.println("- Name of Account Holder = "+ this.acName);
		}
		else
		{
			System.out.println("- Account Number = "+ this.accountNumber);
			System.out.println("- Pin Number = "+ this.pinNumber);
			System.out.println("- Savings Balance = "+ this.savingsBalance);
			System.out.println("- Name of Account Holder = "+ this.acName);
		}
	}
	
}
